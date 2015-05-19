package main

type MediasResponse struct {
	MetaResponse
	PaginationResponse
	Medias []Media `json:"data"`
}

type MetaResponse struct {
	Meta *Meta
}

type PaginationResponse struct {
	Pagination *Pagination
}

type Pagination struct {
	NextUrl string `json:"next_url"`
	NextMaxId string `json:"next_max_id"`
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
