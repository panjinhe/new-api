package types

type ChannelError struct {
	ChannelId   int    `json:"channel_id"`
	ChannelType int    `json:"channel_type"`
	ChannelName string `json:"channel_name"`
	IsMultiKey  bool   `json:"is_multi_key"`
	AutoBan     bool   `json:"auto_ban"`
	UsingKey    string `json:"using_key"`
	UsingKeyIdx int    `json:"using_key_index"`
}

func NewChannelError(channelId int, channelType int, channelName string, isMultiKey bool, usingKey string, usingKeyIdx int, autoBan bool) *ChannelError {
	return &ChannelError{
		ChannelId:   channelId,
		ChannelType: channelType,
		ChannelName: channelName,
		IsMultiKey:  isMultiKey,
		AutoBan:     autoBan,
		UsingKey:    usingKey,
		UsingKeyIdx: usingKeyIdx,
	}
}
