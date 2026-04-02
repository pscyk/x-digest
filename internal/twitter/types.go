package twitter

import (
	"encoding/json"
	"time"
)

// Tweet is the clean output type used for sorting and display.
type Tweet struct {
	ID           string
	Text         string
	AuthorName   string
	AuthorHandle string
	Likes        int
	Retweets     int
	Replies      int
	Quotes       int
	Views        int
	Timestamp    time.Time
	HasMedia     bool
	MediaType    string // "photo", "video", "animated_gif", ""
	IsVerified   bool
	Engagement   int // Likes + Retweets (computed)
}

// --- GraphQL API response types ---

type timelineResponse struct {
	Data struct {
		Home struct {
			HomeTimelineUrt struct {
				Instructions []json.RawMessage `json:"instructions"`
			} `json:"home_timeline_urt"`
		} `json:"home"`
	} `json:"data"`
}

type userByScreenNameResponse struct {
	Data struct {
		User struct {
			Result struct {
				Typename string `json:"__typename"`
				RestID   string `json:"rest_id"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

type userTweetsResponse struct {
	Data struct {
		User struct {
			Result struct {
				Timeline struct {
					Timeline struct {
						Instructions []json.RawMessage `json:"instructions"`
					} `json:"timeline"`
				} `json:"timeline"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

type bookmarksResponse struct {
	Data struct {
		BookmarkTimeline struct {
			Timeline struct {
				Instructions []json.RawMessage `json:"instructions"`
			} `json:"timeline"`
		} `json:"bookmark_timeline_v2"`
	} `json:"data"`
}

type instruction struct {
	Type string `json:"type"`
}

type addEntriesInstruction struct {
	Type    string          `json:"type"`
	Entries []timelineEntry `json:"entries"`
}

type timelineEntry struct {
	EntryID   string          `json:"entryId"`
	SortIndex string          `json:"sortIndex"`
	Content   json.RawMessage `json:"content"`
}

type timelineItemContent struct {
	Typename    string       `json:"__typename"`
	ItemContent tweetContent `json:"itemContent"`
}

type timelineModuleContent struct {
	Typename string `json:"__typename"`
	Items    []struct {
		Item struct {
			ItemContent tweetContent `json:"itemContent"`
		} `json:"item"`
	} `json:"items"`
}

type tweetContent struct {
	Typename     string `json:"__typename"`
	TweetResults struct {
		Result json.RawMessage `json:"result"`
	} `json:"tweet_results"`
}

type tweetResult struct {
	Typename string `json:"__typename"`
	Core     struct {
		UserResults struct {
			Result struct {
				Legacy struct {
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				} `json:"legacy"`
				UserCore struct {
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				} `json:"core"`
				IsBlueVerified bool `json:"is_blue_verified"`
			} `json:"result"`
		} `json:"user_results"`
	} `json:"core"`
	Views struct {
		Count string `json:"count"`
	} `json:"views"`
	Legacy tweetLegacy     `json:"legacy"`
	Tweet  json.RawMessage `json:"tweet,omitempty"`
}

type tweetLegacy struct {
	IDStr         string `json:"id_str"`
	FullText      string `json:"full_text"`
	CreatedAt     string `json:"created_at"`
	FavoriteCount int    `json:"favorite_count"`
	RetweetCount  int    `json:"retweet_count"`
	ReplyCount    int    `json:"reply_count"`
	QuoteCount    int    `json:"quote_count"`
	BookmarkCount int    `json:"bookmark_count"`
	Entities      struct {
		Media []struct {
			Type string `json:"type"`
		} `json:"media"`
	} `json:"entities"`
	ExtendedEntities struct {
		Media []struct {
			Type string `json:"type"`
		} `json:"media"`
	} `json:"extended_entities"`
}
