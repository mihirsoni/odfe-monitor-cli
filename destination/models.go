package destination

// Destination object
type Destination struct {
	ID            string
	Name          string        `json:"name"`
	Type          string        `json:"type"`
	Slack         Slack         `json:"slack,omitempty" yaml:",omitempty"`
	CustomWebhook CustomWebhook `json:"custom_webhook,omitempty" yaml:",omitempty"`
        Sns           Sns           `json:"sns,omitempty" yaml:",omitempty"`
}

// Slack destination object
type Slack struct {
	URL string `json:"url,omitempty" yaml:",omitempty"`
}

// CustomWebhook destination object
type CustomWebhook struct {
	Path         string            `json:"path,omitempty" yaml:",omitempty"`
	HeaderParams map[string]string `json:"header_params,omitempty" yaml:",omitempty"`
	Password     string            `json:"password,omitempty" yaml:",omitempty"`
	Port         int               `json:"port,omitempty" yaml:",omitempty"`
	Scheme       string            `json:"scheme,omitempty" yaml:",omitempty"`
	QueryParams  map[string]string `json:"query_params,omitempty" yaml:",omitempty"`
	Host         string            `json:"host,omitempty" yaml:",omitempty"`
	URL          string            `json:"url,omitempty" yaml:",omitempty"`
	Username     string            `json:"username,omitempty" yaml:",omitempty"`
}

// Sns destination object
type Sns struct {
        SNSTopicARN  string     `json:"topic_arn,omitempty" yaml:",omitempty"`
        IAMroleARN   string     `json:"role_arn,omitempty" yaml:",omitempty"`
}
