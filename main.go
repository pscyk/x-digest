package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	count := flag.Int("count", 20, "number of top tweets to show")
	timeline := flag.String("timeline", "following", "timeline type: 'following' or 'foryou'")
	user := flag.String("user", "", "fetch tweets from a specific user (e.g. elonmusk)")
	profile := flag.String("profile", "", "Firefox profile name override")
	queryID := flag.String("query-id", "", "override GraphQL query ID")
	flag.Parse()

	fmt.Println("Reading Firefox cookies...")
	authToken, ct0, err := extractCookies(*profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var tweets []Tweet

	if *user != "" {
		handle := strings.TrimPrefix(*user, "@")
		fmt.Printf("Fetching tweets from @%s...\n", handle)
		tweets, err = fetchUserTweets(authToken, ct0, handle, *count)
	} else {
		if *timeline != "following" && *timeline != "foryou" {
			fmt.Fprintf(os.Stderr, "Error: --timeline must be 'following' or 'foryou'\n")
			os.Exit(1)
		}
		fmt.Printf("Fetching %s timeline...\n", *timeline)
		tweets, err = fetchTimeline(authToken, ct0, *timeline, *count, *queryID)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	displayTweets(tweets, *count)
}
