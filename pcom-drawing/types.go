package main

type Lengths struct {
	IO    int `json:"io"`
	Maint int `json:"maint"`
	User  int `json:"user"`
	PLC   int `json:"plc"`
	Belt  int `json:"belt"`
	Board int `json:"board"`
}
type Config struct {
	BitCode         string            `json:"bit_code"`
	Role            string            `json:"role"`
	FunctionCode    string            `json:"function_code"`
	UserSerial      string            `json:"user_serial"`
	Maker           string            `json:"maker"`
	FilterCurrent   string            `json:"filter_current"`
	IOConnector     string            `json:"io_connector"`
	MaintConnector  string            `json:"maint_connector"`
	UserConnector   string            `json:"user_connector"`
	BoardConnector  string            `json:"board_connector"`
	BeltConnector   string            `json:"belt_connector"`
	BoardPinOrder   string            `json:"board_pin_order"`
	ConnectorColors map[string]string `json:"connector_colors,omitempty"`
	Lengths         Lengths           `json:"lengths"`
	Revision        string            `json:"revision"`
	Date            string            `json:"date"`
	Notes           string            `json:"notes"`
	PresetID        string            `json:"preset_id"`
}
type Option struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}
type Source struct {
	Filename      string `json:"filename"`
	Slide         int    `json:"slide"`
	DrawingNumber string `json:"drawing_number"`
	Revision      string `json:"revision,omitempty"`
}
type Preset struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Config Config   `json:"config"`
	Source *Source  `json:"source,omitempty"`
	Notes  []string `json:"notes"`
}
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
type SummaryItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
}
type Connector struct {
	Function string `json:"function"`
	Code     string `json:"code"`
	Model    string `json:"model"`
	Pins     int    `json:"pins"`
	Maker    string `json:"maker"`
	Color    string `json:"color"`
	Kind     string `json:"kind"`
	Status   string `json:"status"`
}
type Pin struct {
	Connector string `json:"connector"`
	Model     string `json:"model"`
	Pin       int    `json:"pin"`
	Signal    string `json:"signal"`
	Color     string `json:"color"`
	Target    string `json:"target"`
}
type BOMItem struct {
	Ref      string `json:"ref"`
	Category string `json:"category"`
	Model    string `json:"model"`
	Maker    string `json:"maker"`
	Quantity int    `json:"quantity"`
	Note     string `json:"note"`
}
type Model struct {
	Code         string        `json:"code"`
	Config       Config        `json:"config"`
	Title        string        `json:"title"`
	RoleLabel    string        `json:"role_label"`
	ProfileLabel string        `json:"profile_label"`
	Summary      []SummaryItem `json:"summary"`
	Warnings     []Warning     `json:"warnings"`
	Source       *Source       `json:"source"`
	Connectors   []Connector   `json:"connectors"`
	Pins         []Pin         `json:"pins"`
	BOM          []BOMItem     `json:"bom"`
	SVG          string        `json:"svg"`
}
