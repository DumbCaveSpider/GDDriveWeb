// Go port of https://github.com/SweepSweep2/GDDrive/blob/main/gddrive.py
package main

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	characters   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	startOfLevel = "kS38,1_40_2_125_3_255_11_255_12_255_13_255_4_-1_6_1000_7_1_15_1_18_0_8_1|1_0_2_102_3_255_11_255_12_255_13_255_4_-1_6_1001_7_1_15_1_18_0_8_1|1_0_2_102_3_255_11_255_12_255_13_255_4_-1_6_1009_7_1_15_1_18_0_8_1|1_255_2_255_3_255_11_255_12_255_13_255_4_-1_6_1002_5_1_7_1_15_1_18_0_8_1|1_40_2_125_3_255_11_255_12_255_13_255_4_-1_6_1013_7_1_15_1_18_0_8_1|1_40_2_125_3_255_11_255_12_255_13_255_4_-1_6_1014_7_1_15_1_18_0_8_1|1_0_2_200_3_255_11_255_12_255_13_255_4_-1_6_1005_5_1_7_1_15_1_18_0_8_1|1_0_2_125_3_255_11_255_12_255_13_255_4_-1_6_1006_5_1_7_1_15_1_18_0_8_1|,kA13,0,kA15,0,kA16,0,kA14,,kA6,0,kA7,0,kA25,0,kA17,0,kA18,0,kS39,0,kA2,0,kA3,0,kA8,0,kA4,0,kA9,0,kA10,0,kA22,0,kA23,0,kA24,0,kA27,1,kA40,1,kA48,1,kA41,1,kA42,1,kA28,0,kA29,0,kA31,1,kA32,1,kA36,0,kA43,0,kA44,0,kA45,1,kA46,0,kA47,0,kA33,1,kA34,1,kA35,0,kA37,1,kA38,1,kA39,1,kA19,0,kA26,0,kA20,0,kA21,0,kA11,0;"
	usedKeys     = []int{1, 6, 7, 8, 9, 10, 12, 20, 21, 22, 23, 24, 25, 28, 29, 33, 34, 45, 46, 47, 50, 51, 54, 61, 63, 68, 69, 71, 72, 73, 75, 76, 77, 80, 84, 85, 90, 91, 92, 95, 97, 105, 107, 108, 113, 114, 115}
	gdHeaders    = map[string]string{"User-Agent": ""}
)

// IndexData maps file names to their GD level info
type IndexData map[string]struct {
	LevelID   int    `json:"level_id"`
	LevelName string `json:"level_name"`
}

func log(message string, level int, function ...string) {
	now := time.Now().Format("15:04:05")
	var levelStr string
	switch level {
	case 1:
		levelStr = " [WARNING]"
	case 2:
		levelStr = " [ERROR]"
	}

	fn := ""
	if len(function) > 0 {
		fn = function[0]
	}

	if fn != "" {
		fmt.Printf("[%s]%s %s: %s\n", now, levelStr, fn, message)
	} else {
		fmt.Printf("[%s]%s %s\n", now, levelStr, message)
	}
}

func xorCipher(text string, key string) string {
	result := make([]byte, len(text))
	for i := 0; i < len(text); i++ {
		result[i] = text[i] ^ key[i%len(key)]
	}
	return string(result)
}

func generateGJP2(password string) string {
	salt := "mI29fmAnxgTs"
	h := sha1.New()
	h.Write([]byte(password + salt))
	return hex.EncodeToString(h.Sum(nil))
}

func generateChk(values []string, key string, salt string) string {
	values = append(values, salt)
	combined := strings.Join(values, "")
	h := sha1.New()
	h.Write([]byte(combined))
	hashed := hex.EncodeToString(h.Sum(nil))
	xored := xorCipher(hashed, key)
	return base64.URLEncoding.EncodeToString([]byte(xored))
}

func generateUploadSeed(data string, chars int) string {
	if len(data) < chars {
		return data
	}
	step := len(data) / chars
	var sb strings.Builder
	for i := 0; i < len(data); i += step {
		sb.WriteByte(data[i])
		if sb.Len() >= chars {
			break
		}
	}
	return sb.String()
}

func parseLevel(levelString string) []byte {
	var fileBytes []byte
	parts := strings.Split(levelString, ";")
	if len(parts) < 2 {
		return nil
	}
	levelObjects := parts[1:]

	for _, obj := range levelObjects {
		if obj == "" {
			continue
		}
		splitObj := strings.Split(obj, ",")
		for i := 0; i < len(splitObj); i += 2 {
			if i+1 >= len(splitObj) {
				continue
			}
			keyID, err := strconv.Atoi(splitObj[i])
			if err != nil {
				continue
			}

			isUsed := false
			for _, k := range usedKeys {
				if k == keyID {
					isUsed = true
					break
				}
			}

			if isUsed {
				val, err := strconv.Atoi(splitObj[i+1])
				if err != nil {
					continue
				}
				if keyID != 1 {
					if val > 255 {
						fileBytes = append(fileBytes, 255)
					} else if val < 0 {
						fileBytes = append(fileBytes, 0)
					} else {
						fileBytes = append(fileBytes, byte(val))
					}
				} else {
					if val > 255 {
						fileBytes = append(fileBytes, 255)
					} else if val < 0 {
						fileBytes = append(fileBytes, 0)
					} else {
						fileBytes = append(fileBytes, byte(val-1))
					}
				}
			}
		}
	}
	return fileBytes
}

func makeLevel(fileBytes []byte) (string, int) {
	if len(fileBytes) == 0 {
		return startOfLevel, 0
	}
	currentX := 0
	currentY := 500
	i := 1
	keyOn := 1
	currentObject := fmt.Sprintf("1,%d,2,0,3,500,", int(fileBytes[0])+1)
	levelString := ""
	objectCount := 1

	for i < len(fileBytes) {
		currentObject += fmt.Sprintf("%d,%d", usedKeys[keyOn], fileBytes[i])
		keyOn++

		if keyOn == len(usedKeys) {
			keyOn = 1
			i++
			levelString += currentObject + ";"
			currentY -= 30

			if currentY < 0 {
				currentY = 500
				currentX += 30
			}

			if i >= len(fileBytes) {
				currentObject = ""
				continue
			}

			currentObject = fmt.Sprintf("1,%d,2,%d,3,%d,", int(fileBytes[i])+1, currentX, currentY)
			objectCount++
		} else {
			currentObject += ","
		}
		i++
	}

	if currentObject != "" {
		levelString += currentObject + ";"
		objectCount++
	}

	return startOfLevel + levelString, objectCount
}

func encodeLevel(levelString string, isOfficialLevel bool) string {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte(levelString))
	gw.Close()

	base64Encoded := base64.URLEncoding.EncodeToString(buf.Bytes())
	if isOfficialLevel && len(base64Encoded) > 13 {
		base64Encoded = base64Encoded[13:]
	}
	return base64Encoded
}

func decodeLevel(levelData string, isOfficialLevel bool) string {
	if isOfficialLevel {
		levelData = "H4sIAAAAAAAAA" + levelData
	}

	// Normalise URL-safe base64 to standard
	decoded := strings.ReplaceAll(levelData, "-", "+")
	decoded = strings.ReplaceAll(decoded, "_", "/")
	switch len(decoded) % 4 {
	case 2:
		decoded += "=="
	case 3:
		decoded += "="
	}

	data, err := base64.StdEncoding.DecodeString(decoded)
	if err != nil {
		return ""
	}

	// Try gzip (standard for levels)
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err == nil {
		decompressed, _ := io.ReadAll(gz)
		gz.Close()
		return string(decompressed)
	}

	// Fallback to zlib
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err == nil {
		decompressed, _ := io.ReadAll(r)
		r.Close()
		return string(decompressed)
	}

	return ""
}

func getUserInfo(id int) map[string]string {
	data := url.Values{}
	data.Set("targetAccountID", strconv.Itoa(id))
	data.Set("secret", "Wmfd2893gb7")

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://www.boomlings.com/database/getGJUserInfo20.php", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range gdHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	text := string(body)
	if text == "-1" {
		return nil
	}

	// Format is key:value:key:value...
	parts := strings.Split(text, ":")
	res := make(map[string]string)
	for i := 0; i < len(parts)-1; i += 2 {
		res[parts[i]] = parts[i+1]
	}

	return res
}

// requests a level from the GD database and decodes it
func downloadLevel(id int) string {
	data := url.Values{}
	data.Set("levelID", strconv.Itoa(id))
	data.Set("secret", "Wmfd2893gb7")

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://www.boomlings.com/database/downloadGJLevel22.php", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range gdHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		log("Download request error: "+err.Error(), 2)
		return ""
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	text := string(body)

	if text == "-1" {
		log(fmt.Sprintf("GD server rejected download for level %d (Auth failure?)", id), 2)
		return ""
	}

	splitRes := strings.Split(text, "#")
	if len(splitRes) == 0 || text == "" {
		return ""
	}
	splitRes2 := strings.Split(splitRes[0], ":")

	levelString := ""
	for i := 0; i < len(splitRes2); i += 2 {
		if i+1 < len(splitRes2) && splitRes2[i] == "4" {
			levelString = splitRes2[i+1]
			break
		}
	}

	if levelString == "" {
		log(fmt.Sprintf("Level %d not found!", id), 2)
		return ""
	}
	log(fmt.Sprintf("Downloaded level %d!", id), 0)
	return decodeLevel(levelString, false)
}

func deleteLevel(id int, accountID string, gjp2 string) int {
	data := url.Values{}
	data.Set("accountID", accountID)
	data.Set("gjp2", gjp2)
	data.Set("levelID", strconv.Itoa(id))
	data.Set("secret", "Wmfv2898gc9")

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://www.boomlings.com/database/deleteGJLevelUser20.php", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range gdHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		log("Delete request error: "+err.Error(), 2)
		return 1
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	resBody := string(body)

	if resBody == "1" || resBody == "" || !strings.Contains(resBody, "-1") {
		// GD usually returns "1" on success (or just a non -1 code?)
		// Wait, for deleteGJLevelUser20.php, success is usually NOT -1.
		if resBody != "-1" {
			log(fmt.Sprintf("Level ID %d deleted successfully from GD! (%s)", id, resBody), 0)
			return 0
		}
	}

	log(fmt.Sprintf("Could not delete level ID %d! GD Response: %s", id, resBody), 2)
	return 1
}
