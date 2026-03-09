package conversations

// Conversation represents an intent-to-response mapping for voice interactions.
// Separate from motion scenarios; used by the Speech Layer.
type Conversation struct {
	ID                  string   `json:"id"`
	Intent              string   `json:"intent"` // e.g. find_store
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	ResponseTemplate    string   `json:"response_template"`    // e.g. "{{brand}} store is on the {{floor}} floor"
	ResponseProviderURL string   `json:"response_provider_url,omitempty"` // optional; for dynamic responses
	SupportedLanguages  []string `json:"supported_languages"` // uz, en, ru, az, ar
	TenantID            string   `json:"tenant_id,omitempty"`
}
