package monitor

// Search hello
type Search struct {
	Search struct {
		Indices []string               `json:"indices"`
		Query   map[string]interface{} `json:"query"`
	} `json:"search"`
}

type Trigger struct {
	ID       string `json:"id,omitempty" yaml:"-"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	//YCondition (YAML Condition), is to to minimize customer input for now
	YCondition string    `json:"-" yaml:"condition"`
	Condition  Condition `json:"condition" yaml:"-"`
	Actions    []Action  `json:"actions,omitempty"`
}

// Period hello
type Period struct {
	Interval int    `json:"interval"`
	Unit     string `json:"unit"`
}

// Cron hello
type Cron struct {
	Expression string `json:"expression"`
	Timezone   string `json:"timezone"`
}

// Schedule world
type Schedule struct {
	Period *Period `json:"period,omitempty"`
	Cron   *Cron   `json:"cron,omitempty"`
}

//Action action model
type Action struct {
	ID            string `json:"id,omitempty" yaml:"-"`
	Name          string `json:"name"`
	DestinationID string `json:"destination_id,omitempty" yaml:"destinationId"`
	// Why duplicated Subject and Message ? This is to simplify customer experience on writing new monitors.
	// Taking input which is important default are being filled by CLI
	Subject         string `json:"-"`
	Message         string `json:"-"`
	SubjectTemplate Script `json:"subject_template" yaml:"-"`
	MessageTemplate Script `json:"message_template" yaml:"-"`
}

type Script struct {
	Source string `json:"source"`
	Lang   string `json:"lang"`
}
type Condition struct {
	Script Script `json:"script"`
}

// Monitor nice
type Monitor struct {
	primaryTerm string // Required for Update
	seqNo       string // Required for Update
	id          string
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Enabled     bool      `json:"enabled"`
	Schedule    Schedule  `json:"schedule"`
	Inputs      []Search  `json:"inputs"`
	Triggers    []Trigger `json:"triggers"`
}
