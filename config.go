package main

type Config struct {
    Colors struct {
        White       string `json:"white"`
        Teal        string `json:"teal"`
        Highlight   string `json:"highlight"`
        Command     string `json:"command"`
        Blinking    string `json:"blinking"`
        Label       string `json:"label"`
        Value       string `json:"value"`
        Focused     string `json:"focused"`
    } `json:"colors"`
    Navigation struct {
        Up    string `json:"up"`
        Down  string `json:"down"`
        Left  string `json:"left"`
        Right string `json:"right"`
    } `json:"navigation"`
}