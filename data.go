package main

type MediasResponse struct {
	MetaResponse
	Medias []Media `json:"data"`
}

type MetaResponse struct {
	Meta *Meta
}

type Meta struct {
	Code         int
	ErrorType    string `json:"error_type"`
	ErrorMessage string `json:"error_message"`
}

type Media struct {
	Images *Images
}

type Images struct {
	LowResolution      *Image `json:"low_resolution"`
	Thumbnail          *Image
	StandardResolution *Image `json:"standard_resolution"`
}

type Image struct {
	Url    string
	Width  int64
	Height int64
}
