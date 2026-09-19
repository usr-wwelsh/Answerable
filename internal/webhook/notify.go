package webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 10 * time.Second}

// Notification is the intake-request summary sent to a chat webhook.
// ConfirmURL and DenyURL are rendered as hyperlink text rather than shown
// as raw URLs, since Discord/Slack chat is where a human actually reads
// and acts on these.
type Notification struct {
	Name, Contact, Need string
	ConfirmURL, DenyURL string
}

func Notify(url string, n Notification) error {
	summary := fmt.Sprintf("New intake request from %s (%s): %s", n.Name, n.Contact, n.Need)

	// Slack/mrkdwn masked-link syntax: <url|label>.
	slackText := fmt.Sprintf("%s\n<%s|Confirm> | <%s|Deny>", summary, n.ConfirmURL, n.DenyURL)

	payload, err := json.Marshal(map[string]any{
		"content": summary,
		"text":    slackText,
		// Discord ignores unknown top-level fields, so Slack/Teams
		// endpoints simply skip this. Masked links only render inside
		// embeds in Discord, not in plain "content".
		"embeds": []map[string]any{
			{
				"description": summary,
				"fields": []map[string]any{
					{"name": "Respond", "value": fmt.Sprintf("[Confirm](%s) · [Deny](%s)", n.ConfirmURL, n.DenyURL)},
				},
			},
		},
	})
	if err != nil {
		return err
	}

	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
