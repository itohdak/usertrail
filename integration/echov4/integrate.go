package echov4

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var (
	userPrevActivity sync.Map
	userActivity     sync.Map

	pattern []string
)

// パターンごとにマッチした部分をそのまま置換する関数
func replaceWithMatchedPattern(input string, patterns []string) (string, error) {
	// 結果を保持する文字列
	result := input

	for _, pattern := range patterns {
		// パターンをコンパイル
		re, err := regexp.Compile(pattern)
		if err != nil {
			return "", fmt.Errorf("正規表現のコンパイルに失敗しました: %v", err)
		}

		// マッチした部分をそのまま置換
		result = re.ReplaceAllString(result, pattern)
	}

	return result, nil
}

func myMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		currMethod := c.Request().Method
		rawURL := c.Request().URL.Path
		currURL, _ := replaceWithMatchedPattern(rawURL, pattern)
		cookie, err := c.Cookie("my_session_id")
		sid := cookie.Value
		if err != nil {
			uuid, _ := uuid.NewRandom()
			c.SetCookie(&http.Cookie{
				Name:  "my_session_id",
				Value: fmt.Sprintf("%s", uuid),
			})
			sid = fmt.Sprintf("%s", uuid)
		} else {
			if prevActivityCached, found := userPrevActivity.Load(sid); found {
				prevActivity := prevActivityCached.(struct {
					Method string
					URL    string
				})
				trans := fmt.Sprintf("%s %s --> %s %s", prevActivity.Method, prevActivity.URL, currMethod, currURL)
				cnt, _ := userActivity.LoadOrStore(trans, int64(0))
				userActivity.Store(trans, cnt.(int64)+1)
			}
		}
		userPrevActivity.Store(sid, struct {
			Method string
			URL    string
		}{
			Method: currMethod,
			URL:    currURL,
		})
		if err = next(c); err != nil {
			c.Error(err)
		}
		return nil
	}
}

func Integrate(e *echo.Echo, matchPatternList []string) {
	pattern = matchPatternList
	e.Use(myMiddleware)
	go func() {
		for {
			printMermaid()
			time.Sleep(1 * time.Minute)
		}
	}()
}

// matchPatternList example
// []string{
// 	`^/api/initialize$`,
// 	`^/api/tag$`,
// 	`^/api/user/[0-9a-zA-Z]+/theme$`,
// 	`^/api/livestream/reservation$`,
// 	`^/api/livestream/search$`,
// 	`^/api/livestream$`,
// 	`^/api/user/[0-9a-zA-Z]+/livestream$`,
// 	`^/api/livestream/[0-9a-zA-Z]+$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/livecomment$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/livecomment$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/reaction$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/reaction$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/report$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/ngwords$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/livecomment/[0-9a-zA-Z]+/report$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/moderate$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/enter$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/exit$`,
// 	`^/api/register$`,
// 	`^/api/login$`,
// 	`^/api/user/me$`,
// 	`^/api/user/[0-9a-zA-Z]+$`,
// 	`^/api/user/[0-9a-zA-Z]+/statistics$`,
// 	`^/api/user/[0-9a-zA-Z]+/icon$`,
// 	`^/api/icon$`,
// 	`^/api/livestream/[0-9a-zA-Z]+/statistics$`,
// 	`^/api/payment$`,
// }

func printMermaid() {
	m := map[string]int64{}
	userActivity.Range(func(key, value interface{}) bool {
		m[fmt.Sprint(key)] = value.(int64)
		return true
	})
	userStateStrings := sync.Map{}
	var mermaidString string
	mermaidString += "```mermaid\ngraph LR\n"
	var maxCount int64 = 0
	for _, cnt := range m {
		maxCount = max(maxCount, cnt)
	}
	var id int64 = 0
	for trans, cnt := range m {
		if cnt <= 1 {
			continue
		}
		transList := strings.Split(trans, " --> ")
		from := transList[0]
		to := transList[1]
		var fromString string
		var toString string
		if str, found := userStateStrings.Load(from); found {
			fromString = str.(string)
		} else {
			uuid, _ := uuid.NewRandom()
			fromString = fmt.Sprintf("%s[\"%s\"]", uuid, from)
			userStateStrings.Store(from, fromString)
		}
		if str, found := userStateStrings.Load(to); found {
			toString = str.(string)
		} else {
			uuid, _ := uuid.NewRandom()
			toString = fmt.Sprintf("%s[\"%s\"]", uuid, to)
			userStateStrings.Store(to, toString)
		}
		mermaidString += fmt.Sprintf(
			"  %s -->|%d| %s\n",
			fromString,
			cnt,
			toString,
		)
		mermaidString += fmt.Sprintf(
			"  linkStyle %d stroke-width:%.1fpx;\n",
			id,
			1+float32(29)*float32(cnt)/float32(maxCount),
		)
		id++
	}
	mermaidString += "```"
	log.Printf("%s", mermaidString)
}
