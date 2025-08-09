package telegram

// Minimal Telegram Update types needed for webhook handling

type Update struct {
    UpdateID int     `json:"update_id"`
    Message  *MessageIn `json:"message"`
}

type MessageIn struct {
    MessageID int    `json:"message_id"`
    From      *UserIn `json:"from"`
    Chat      *Chat   `json:"chat"`
    Date      int64  `json:"date"`
    Text      string `json:"text"`
}

type UserIn struct {
    ID           int64  `json:"id"`
    IsBot        bool   `json:"is_bot"`
    FirstName    string `json:"first_name"`
    LastName     string `json:"last_name"`
    Username     string `json:"username"`
    LanguageCode string `json:"language_code"`
}

type Chat struct {
    ID   int64  `json:"id"`
    Type string `json:"type"`
}


