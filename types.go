package main

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

// --- X GraphQL API response types ---

type TimelineResponse struct {
	Data struct {
		Home struct {
			HomeTimelineUrt struct {
				Instructions []json.RawMessage `json:"instructions"`
			} `json:"home_timeline_urt"`
		} `json:"home"`
	} `json:"data"`
}

type UserByScreenNameResponse struct {
	Data struct {
		User struct {
			Result struct {
				Typename string `json:"__typename"`
				RestID   string `json:"rest_id"`
				Legacy   struct {
					Name       string `json:"name"`
					ScreenName string `json:"screen_name"`
				} `json:"legacy"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

type UserTweetsResponse struct {
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

type Instruction struct {
	Type string `json:"type"`
}

type AddEntriesInstruction struct {
	Type    string          `json:"type"`
	Entries []TimelineEntry `json:"entries"`
}

type TimelineEntry struct {
	EntryID   string          `json:"entryId"`
	SortIndex string          `json:"sortIndex"`
	Content   json.RawMessage `json:"content"`
}

type TypedContent struct {
	Typename string `json:"__typename"`
}

type TimelineItemContent struct {
	Typename    string       `json:"__typename"`
	ItemContent TweetContent `json:"itemContent"`
}

type TimelineModuleContent struct {
	Typename string `json:"__typename"`
	Items    []struct {
		Item struct {
			ItemContent TweetContent `json:"itemContent"`
		} `json:"item"`
	} `json:"items"`
}

type TweetContent struct {
	Typename     string `json:"__typename"`
	TweetResults struct {
		Result json.RawMessage `json:"result"`
	} `json:"tweet_results"`
}

type TweetResult struct {
	Typename string `json:"__typename"`
	Core struct {
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
	Legacy TweetLegacy     `json:"legacy"`
	Tweet  json.RawMessage `json:"tweet,omitempty"` // For TweetWithVisibilityResults
}

type TweetLegacy struct {
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
