package utils

var CategoryMap = map[string]string{
	"1":  "Film & Animation",
	"2":  "Autos & Vehicles",
	"10": "Music",
	"15": "Pets & Animals",
	"17": "Sports",
	"18": "Short Movies",
	"19": "Travel & Events",
	"20": "Gaming",
	"21": "Videoblogging",
	"22": "People & Blogs",
	"23": "Comedy",
	"24": "Entertainment",
	"25": "News & Politics",
	"26": "Howto & Style",
	"27": "Education",
	"28": "Science & Technology",
}

var SupportedRegions = []string{"IN","US","DE"}

/*curl commands =>

curl "http://localhost:8080/regions"

curl "http://localhost:8080/categories?region=US"

curl "http://localhost:8080/videos?region=IN&category=10&maxResults=5"

curl "http://localhost:8080/trending?maxResults=7"

curl "http://localhost:8080/search?query=india%20election&region=US"

curl "http://localhost:8080/comments?videoId=VIDEO_ID"

curl "http://localhost:8080/videostats?videoId=VIDEO_ID"




*/
