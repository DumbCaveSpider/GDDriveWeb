package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

// --- Request / Response structs ---

type LoginInitRequest struct {
	AccountID string `json:"account_id"`
}

type LoginValidateRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	AccountID      string `json:"account_id"`
	ValidationCode string `json:"validation_code"`
}

type FileNameRequest struct {
	FileName string `json:"file_name"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

type FileInfo struct {
	Name      string `json:"name"`
	LevelID   int    `json:"level_id"`
	LevelName string `json:"level_name"`
}

// reqCreds holds credentials extracted from request headers.
type reqCreds struct {
	Username  string
	GJP2      string
	AccountID string
}

func (c reqCreds) valid() bool {
	return c.Username != "" && c.GJP2 != "" && c.AccountID != ""
}

func getCreds(r *http.Request) reqCreds {
	return reqCreds{
		Username:  r.Header.Get("X-GD-Username"),
		GJP2:      r.Header.Get("X-GD-GJP2"),
		AccountID: r.Header.Get("X-GD-AccountID"),
	}
}

// --- Database state ---

var (
	db                 *sql.DB
	pendingValidations = make(map[string]string)
	pvLock             sync.Mutex
)

func initDB() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		dsn = "root@tcp(127.0.0.1:3306)/gddrive?parseTime=true&multiStatements=true"
	}

	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log("Failed to open MySQL connection: "+err.Error(), 2)
		os.Exit(1)
	}

	// Verify the connection
	if err := db.Ping(); err != nil {
		log("Failed to connect to MySQL/MariaDB server: "+err.Error(), 2)
		log("Make sure the database exists and your credentials (DB_URL) are correct.", 2)
		os.Exit(1)
	}

	schema, err := os.ReadFile("schema.sql")
	if err != nil {
		log("Failed to read schema.sql: "+err.Error(), 2)
	} else {
		_, err = db.Exec(string(schema))
		if err != nil {
			log("Failed to execute schema (MySQL compatibility): "+err.Error(), 2)
		}
	}

	// Migration from index.json
	migrateIndex()
}

func migrateIndex() {
	data, err := os.ReadFile("index.json")
	if err == nil {
		var index map[string]struct {
			LevelID   int    `json:"level_id"`
			LevelName string `json:"level_name"`
		}
		if json.Unmarshal(data, &index) == nil && len(index) > 0 {
			log("Migrating index.json to database...", 0)
			for name, info := range index {
				// We don't know the account id, so we skip if no accounts exist or use a default
				// For now, only migrate if at least one account exists? No, let's keep it in files with accountId 0 if necessary.
				_, err := db.Exec("INSERT IGNORE INTO files (fileName, levelId, levelName, accountId) VALUES (?, ?, ?, 0)",
					name, info.LevelID, info.LevelName)
				if err != nil {
					log("Migration error for "+name+": "+err.Error(), 2)
				}
			}
			// Rename or delete index.json to avoid multiple migrations
			os.Rename("index.json", "index.json.bak")
		}
	}
}

// --- Middleware ---

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-GD-Username, X-GD-GJP2, X-GD-AccountID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

// --- Handlers ---

func handleStatus(w http.ResponseWriter, r *http.Request) {
	creds := getCreds(r)
	respondJSON(w, 200, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"logged_in": creds.valid(),
			"username":  creds.Username,
		},
	})
}

// POST /api/login/init — starts validation by returning a code
func handleLoginInit(w http.ResponseWriter, r *http.Request) {
	var req LoginInitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}

	if req.AccountID == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Account ID is required"})
		return
	}

	var exists int
	err := db.QueryRow("SELECT 1 FROM accounts WHERE accountId = ?", req.AccountID).Scan(&exists)
	requiresVerification := (err == sql.ErrNoRows)

	var code string
	if requiresVerification {
		code = generateRandomCode(8)
		pvLock.Lock()
		pendingValidations[req.AccountID] = code
		pvLock.Unlock()
	}

	respondJSON(w, 200, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"validation_code":       code,
			"requires_verification": requiresVerification,
		},
	})
}

// POST /api/login/validate — checks GD user profile for the code and saves credentials
func handleLoginValidate(w http.ResponseWriter, r *http.Request) {
	var req LoginValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" || req.AccountID == "" {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Username, Password, and Account ID are required"})
		return
	}

	targetID, err := strconv.Atoi(req.AccountID)
	if err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid numeric Account ID"})
		return
	}

	// Check if account already exists
	var storedGJP2 string
	err = db.QueryRow("SELECT gjp2 FROM accounts WHERE accountId = ?", targetID).Scan(&storedGJP2)
	isNew := (err == sql.ErrNoRows)

	// If it's a new account, we MUST perform validation
	if isNew {
		if req.ValidationCode == "" {
			respondJSON(w, 400, APIResponse{Success: false, Message: "Verification code is required for new accounts"})
			return
		}

		info := getUserInfo(targetID)
		if info == nil {
			respondJSON(w, 404, APIResponse{Success: false, Message: "Geometry Dash account not found"})
			return
		}

		if info["61"] != req.ValidationCode {
			respondJSON(w, 401, APIResponse{
				Success: false,
				Message: fmt.Sprintf("Validation failed. Check if you set the token at the Custom Field in your GD Profile."),
			})
			return
		}
	} else {
		// Existing account — verify the submitted password matches stored GJP2
		submittedGJP2 := generateGJP2(req.Password)
		if submittedGJP2 != storedGJP2 {
			respondJSON(w, 401, APIResponse{Success: false, Message: "Invalid credentials"})
			return
		}
	}

	// Success! Generate GJP2 and save/update user in DB
	gjp2 := generateGJP2(req.Password)
	if isNew {
		_, err = db.Exec(
			"INSERT INTO accounts (accountId, username, gjp2) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE username=VALUES(username), gjp2=VALUES(gjp2)",
			targetID, req.Username, gjp2)
		if err != nil {
			respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to save account: " + err.Error()})
			return
		}
	}

	log("Validated and saved user: "+req.Username, 0)

	msg := "Login successful!"
	if isNew {
		msg = "Account validated and logged in!"
	}

	respondJSON(w, 200, APIResponse{
		Success: true,
		Message: msg,
		Data: map[string]string{
			"username":   req.Username,
			"gjp2":       gjp2,
			"account_id": req.AccountID,
		},
	})
}

func handleLogout(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, 200, APIResponse{Success: true, Message: "Logged out successfully"})
}

func handleListFiles(w http.ResponseWriter, r *http.Request) {
	creds := getCreds(r)
	// We allow listing all for now, or filter by user?
	// User said "accounts who owns that account". Let's filter by the requesting user.
	if !creds.valid() {
		respondJSON(w, 401, APIResponse{Success: false, Message: "Login required to list files"})
		return
	}

	rows, err := db.Query("SELECT fileName, levelId, levelName FROM files WHERE accountId = ?", creds.AccountID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Database query error"})
		return
	}
	defer rows.Close()

	var files []FileInfo
	for rows.Next() {
		var f FileInfo
		rows.Scan(&f.Name, &f.LevelID, &f.LevelName)
		files = append(files, f)
	}

	respondJSON(w, 200, APIResponse{Success: true, Data: files})
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	creds := getCreds(r)
	if !creds.valid() {
		respondJSON(w, 401, APIResponse{Success: false, Message: "Not logged in"})
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Could not parse form: " + err.Error()})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Missing file field"})
		return
	}
	defer file.Close()

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Could not read file"})
		return
	}

	fileName := header.Filename

	log("Uploading level...", 0)

	levelString, objCount := makeLevel(fileBytes)
	encodedLevel := encodeLevel(levelString, false)

	// Get existing level Name if any
	var levelName string
	row := db.QueryRow("SELECT levelName FROM files WHERE fileName = ? AND accountId = ?", fileName, creds.AccountID)
	err = row.Scan(&levelName)
	if err == sql.ErrNoRows {
		levelName = generateRandomCode(20)
	}

	formData := url.Values{}
	formData.Set("gameVersion", "22")
	formData.Set("binaryVersion", "47")
	formData.Set("accountID", creds.AccountID)
	formData.Set("gjp2", creds.GJP2)
	formData.Set("userName", creds.Username)
	formData.Set("levelID", "0")
	formData.Set("levelName", levelName)
	formData.Set("levelDesc", "")
	formData.Set("levelVersion", "1")
	formData.Set("levelLength", "0")
	formData.Set("audioTrack", "0")
	formData.Set("auto", "0")
	formData.Set("password", "1")
	formData.Set("original", "0")
	formData.Set("twoPlayer", "0")
	formData.Set("songID", "645828")
	formData.Set("objects", strconv.Itoa(objCount))
	formData.Set("coins", "0")
	formData.Set("requestedStars", "10")
	formData.Set("unlisted", "2")
	formData.Set("ldm", "0")
	formData.Set("levelString", encodedLevel)
	formData.Set("seed2", generateChk([]string{generateUploadSeed(encodedLevel, 50)}, "41274", "xI25fpAapCQg"))
	formData.Set("secret", "Wmfd2893gb7")
	formData.Set("dvs", "3")

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://www.boomlings.com/database/uploadGJLevel21.php", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "")

	resp, err := client.Do(req)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Upload request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	resBody, _ := io.ReadAll(resp.Body)
	resText := string(resBody)

	if resText == "-1" {
		log("Failed to upload level!", 2)
		respondJSON(w, 500, APIResponse{Success: false, Message: "GD server rejected the upload. Check your credentials."})
		return
	}

	levelID, _ := strconv.Atoi(strings.TrimSpace(resText))

	log(fmt.Sprintf("Level successfully uploaded! ID: %d", levelID), 0)

	_, err = db.Exec("REPLACE INTO files (fileName, levelId, levelName, accountId) VALUES (?, ?, ?, ?)",
		fileName, levelID, levelName, creds.AccountID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to update file index: " + err.Error()})
		return
	}

	respondJSON(w, 200, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Uploaded successfully! Level ID: %d", levelID),
		Data:    map[string]interface{}{"level_id": levelID, "file_name": fileName},
	})
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	var req FileNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}

	var levelId int
	err := db.QueryRow("SELECT levelId FROM files WHERE fileName = ?", req.FileName).Scan(&levelId)
	if err == sql.ErrNoRows {
		respondJSON(w, 404, APIResponse{Success: false, Message: "File not found"})
		return
	}

	levelString := downloadLevel(levelId)
	if levelString == "" {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to download level from GD servers"})
		return
	}

	fileBytes := parseLevel(levelString)

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, req.FileName))
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	w.WriteHeader(200)
	w.Write(fileBytes)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	creds := getCreds(r)
	if !creds.valid() {
		respondJSON(w, 401, APIResponse{Success: false, Message: "Not logged in"})
		return
	}

	var req FileNameRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, APIResponse{Success: false, Message: "Invalid request body"})
		return
	}

	var levelId int
	err := db.QueryRow("SELECT levelId FROM files WHERE fileName = ? AND accountId = ?", req.FileName, creds.AccountID).Scan(&levelId)
	if err == sql.ErrNoRows {
		respondJSON(w, 404, APIResponse{Success: false, Message: "File not found or access denied"})
		return
	}

	result := deleteLevel(levelId, creds.AccountID, creds.GJP2)
	if result != 0 {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to delete level from GD servers"})
		return
	}

	_, err = db.Exec("DELETE FROM files WHERE fileName = ? AND accountId = ?", req.FileName, creds.AccountID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to delete from local database"})
		return
	}

	respondJSON(w, 200, APIResponse{
		Success: true,
		Message: fmt.Sprintf("'%s' deleted successfully!", req.FileName),
	})
}

func handleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	creds := getCreds(r)
	if !creds.valid() {
		respondJSON(w, 401, APIResponse{Success: false, Message: "Not logged in"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Database transaction error"})
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM files WHERE accountId = ?", creds.AccountID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to delete files index"})
		return
	}

	_, err = tx.Exec("DELETE FROM accounts WHERE accountId = ?", creds.AccountID)
	if err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to delete account from database"})
		return
	}

	if err := tx.Commit(); err != nil {
		respondJSON(w, 500, APIResponse{Success: false, Message: "Failed to commit deletion"})
		return
	}

	log(fmt.Sprintf("Account %s (%s) deleted successfully.", creds.Username, creds.AccountID), 0)
	respondJSON(w, 200, APIResponse{
		Success: true,
		Message: "Account and all indexed files deleted successfully.",
	})
}

// --- Utils ---

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func startServer() {
	initDB()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", corsMiddleware(handleStatus))
	mux.HandleFunc("/api/login/init", corsMiddleware(handleLoginInit))
	mux.HandleFunc("/api/login/validate", corsMiddleware(handleLoginValidate))
	mux.HandleFunc("/api/logout", corsMiddleware(handleLogout))
	mux.HandleFunc("/api/files", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleListFiles(w, r)
		case http.MethodDelete:
			handleDelete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/files/upload", corsMiddleware(handleUpload))
	mux.HandleFunc("/api/files/download", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleDownload(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/account", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			handleDeleteAccount(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	fmt.Println("Welcome to GDDrive!")

	// Serve static files from the "dist" directory if it exists
	distPaths := []string{"./dist", "../dist"}
	for _, p := range distPaths {
		if _, err := os.Stat(p); err == nil {
			mux.Handle("/", http.FileServer(http.Dir(p)))
			log("GDDrive Server: Serving frontend from "+p, 0)
			break
		}
	}

	log("GDDrive Server running on :3002", 0)
	http.ListenAndServe(":3002", mux)
}

func main() {
	if err := godotenv.Load(); err == nil {
		log(".env file loaded successfully", 0)
	} else {
		// Try root too if not found (since we might run from root)
		godotenv.Load("../.env")
	}
	rand.Seed(time.Now().UnixNano())
	startServer()
}
