package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"text/template"
	"time"

	"github.com/gorilla/mux"
)

type GrafanaAlert struct {
	Title       string `json:"title"`
	RuleID      int    `json:"ruleId"`
	RuleName    string `json:"ruleName"`
	RuleURL     string `json:"ruleUrl"`
	State       string `json:"state"`
	ImageURL    string `json:"imageUrl"`
	Message     string `json:"message"`
	EvalMatches []struct {
		Metric string `json:"metric"`
		Value  int    `json:"value"`
	} `json:"evalMatches"`
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	token := vars["token"]
	chatID := vars["chatId"]
	var alert GrafanaAlert
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		reqData, e := ioutil.ReadAll(r.Body)
		if e != nil {
			return
		} else {
			json.Unmarshal(reqData, &alert)
			urlTelegram := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
			templateAlert := `
[{{.State}}] {{.Title}}
state: {{.RuleName}}
message: {{.Message}}
URL: {{.RuleURL}}

Metrics:
{{range .EvalMatches}}
	{{.Metric}}: {{.Value}}
{{end}}
			`
			t, _ := template.New("alert").Parse(templateAlert)

			var message bytes.Buffer
			t.Execute(&message, alert)
			form := url.Values{
				"chat_id": {chatID},
				"text":    {message.String()},
			}
			body := bytes.NewBufferString(form.Encode())
			_, err := http.Post(urlTelegram, "application/x-www-form-urlencoded", body)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	fmt.Fprintf(w, "ok")

}

func main() {

	port := flag.String("port", "17575", "Listen port")

	flag.Parse()

	listenAddr := ":" + *port

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Listen on", listenAddr)

	r := mux.NewRouter()

	r.HandleFunc("/webhooktel/{token}/{chatId}", handleWebhook)

	srv := &http.Server{
		Addr:         listenAddr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}
}
