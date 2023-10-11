package main

type course struct {
	Author_name  string   `json:"author_name"` // 虽然在一个包内，但是序列化要用到, 故首字母大写
	Column_sku   int32    `json:"column_sku"`
	Unit         string   `json:"unit"`
	Author_intro string   `json:"author_intro"`
	Count        int32    `json:"count"`
	Count_req    int32    `json:"count_req"`
	Is_finlish   bool     `json:"is_finlish"`
	Title        string   `json:"title"`
	Lables       []string `json:"lables"`
}

type article struct {
	Article_id    int32  `json:"article_id"`
	Column_sku    int32  `json:"column_sku"`
	Article_title string `json:"article_title"`
	Chapter_Name  string `json:"chapter_Name"`
	Chapter_id    string `json:"chapter_id"`
	Article_index int32  `json:"article_index"`
}

type article_content struct {
	Article_content string `json:"article_content"`
	Article_ctime   int32  `json:"article_ctime"`
	Article_title   string `json:"article_title"`
	Audio_dubber    string `json:"audio_dubber"`
	Audio_size      int32  `json:"audio_size"`
	Audio_time      string `json:"audio_time"`
	Audio_title     string `json:"audio_title"`
	Audio_url       string `json:"audio_url"`
	Author_name     string `json:"author_name"`
	Chapter_id      string `json:"chapter_id"`
	Chapter_name    string `json:"chapter_name"`
	Video_size      int32  `json:"video_size"`
	Video_time      string `json:"video_time"`
}
