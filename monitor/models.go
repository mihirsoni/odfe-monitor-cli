/*
 * Copyright 2020 Amazon.com, Inc. or its affiliates. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License").
 * You may not use this file except in compliance with the License.
 * A copy of the License is located at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * or in the "license" file accompanying this file. This file is distributed
 * on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
 * express or implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */

package monitor

// Search hello
type Search struct {
	Search struct {
		Indices []string               `json:"indices"`
		Query   map[string]interface{} `json:"query"`
	} `json:"search"`
}

//Trigger define a Trigger struct
type Trigger struct {
	ID       string `json:"id,omitempty" yaml:"-"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	//YCondition (YAML Condition), is to to minimize customer input for now
	YCondition string    `json:"-" yaml:"condition"`
	Condition  Condition `json:"condition" yaml:"-"`
	Actions    []Action  `json:"actions,omitempty"`
}

// Period Define monitor with period
type Period struct {
	Interval int    `json:"interval"`
	Unit     string `json:"unit"`
}

// Cron Define monitor with Cron
type Cron struct {
	Expression string `json:"expression"`
	Timezone   string `json:"timezone"`
}

// Schedule type of Monitor (Cron / Period)
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

//Script Works for mustache and painless
type Script struct {
	Source string `json:"source"`
	Lang   string `json:"lang"`
}

//Condition define condition for the triggers
type Condition struct {
	Script Script `json:"script"`
}

// Monitor Alert monitor object
type Monitor struct {
	primaryTerm string // Required for Update
	seqNo       string // Required for Update
	id          string
	Name        string    `json:"name"`
	Enabled     bool      `json:"enabled"`
	Schedule    Schedule  `json:"schedule"`
	Inputs      []Search  `json:"inputs"`
	Triggers    []Trigger `json:"triggers"`
}
