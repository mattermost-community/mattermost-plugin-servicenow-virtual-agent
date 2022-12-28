package serializer

type VirtualAgentRequestBody struct {
	Action    string       `json:"action"`
	Message   *MessageBody `json:"message"`
	RequestID string       `json:"requestId"`
	UserID    string       `json:"userId"`
}

type MessageBody struct {
	Attachment *MessageAttachment `json:"attachment"`
	Text       string             `json:"text"`
	Typed      bool               `json:"typed"`
}

type MessageAttachment struct {
	URL         string `json:"url"`
	ContentType string `json:"contentType"`
	FileName    string `json:"fileName"`
}

type OutputText struct {
	UIType   string `json:"uiType"`
	Group    string `json:"group"`
	Value    string `json:"value"`
	ItemType string `json:"type"`
	MaskType string `json:"maskType"`
	Label    string `json:"label"`
	Required bool   `json:"required"`
}

type OutputLinkValue struct {
	Action string `json:"action"`
}

type OutputLink struct {
	UIType        string `json:"uiType"`
	Group         string `json:"group"`
	Label         string `json:"label"`
	Header        string `json:"header"`
	Type          string `json:"type"`
	Value         OutputLinkValue
	PromptMessage string `json:"promptMsg"`
}

type GroupedPartsOutputControl struct {
	UIType string                           `json:"uiType"`
	Group  string                           `json:"group"`
	Header string                           `json:"header"`
	Type   string                           `json:"type"`
	Values []GroupedPartsOutputControlValue `json:"values"`
}

type GroupedPartsOutputControlValue struct {
	Label       string `json:"label"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type TopicPickerControl struct {
	UIType         string   `json:"uiType"`
	Group          string   `json:"group"`
	NLUTextEnabled bool     `json:"nluTextEnabled"`
	PromptMessage  string   `json:"promptMsg"`
	Label          string   `json:"label"`
	Options        []Option `json:"options"`
}

type OutputCard struct {
	UIType       string `json:"uiType"`
	Group        string `json:"group"`
	Data         string `json:"data"`
	TemplateName string `json:"templateName"`
}

type OutputCardRecordData struct {
	SysID            string          `json:"sys_id"`
	Subtitle         string          `json:"subtitle"`
	DataNowSmartLink string          `json:"dataNowSmartLink"`
	Title            string          `json:"title"`
	Fields           []*RecordFields `json:"fields"`
	TableName        string          `json:"table_name"`
	URL              string          `json:"url"`
	Target           string          `json:"target"`
}

type OutputCardVideoData struct {
	Link             string `json:"link"`
	Description      string `json:"description"`
	ID               string `json:"id"`
	DataNowSmartLink string `json:"dataNowSmartLink"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	Target           string `json:"target"`
}

type OutputCardImageData struct {
	Image            string `json:"image"`
	Description      string `json:"description"`
	DataNowSmartLink string `json:"dataNowSmartLink"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	ImageAlt         string `json:"imageAlt"`
	Target           string `json:"target"`
}

type RecordFields struct {
	FieldLabel string `json:"fieldLabel"`
	FieldValue string `json:"fieldValue"`
}

type Picker struct {
	UIType         string   `json:"uiType"`
	Group          string   `json:"group"`
	Required       bool     `json:"required"`
	NLUTextEnabled bool     `json:"nluTextEnabled"`
	Label          string   `json:"label"`
	ItemType       string   `json:"itemType"`
	Options        []Option `json:"options"`
	Style          string   `json:"style"`
	MultiSelect    bool     `json:"multiSelect"`
}

type Option struct {
	Label       string `json:"label"`
	Value       string `json:"value"`
	Enabled     bool   `json:"enabled"`
	Description string `json:"description"`
	Attachment  string `json:"attachment"`
}

type OutputImage struct {
	UIType  string `json:"uiType"`
	Group   string `json:"group"`
	Value   string `json:"value"`
	AltText string `json:"altText"`
}

type DefaultDate struct {
	UIType         string `json:"uiType"`
	Group          string `json:"group"`
	Required       bool   `json:"required"`
	NLUTextEnabled bool   `json:"nluTextEnabled"`
	Label          string `json:"label"`
}
