package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"x-digest/internal/cookies"
	"x-digest/internal/display"
	"x-digest/internal/twitter"
)

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil && !errors.Is(err, flag.ErrHelp) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, out, errOut io.Writer) error {
	flags := flag.NewFlagSet("x-digest", flag.ContinueOnError)
	flags.SetOutput(errOut)
	count := flags.Int("count", 20, "number of top tweets to show (1–100)")
	timeline := flags.String("timeline", "following", "timeline type: 'following' or 'foryou'")
	user := flags.String("user", "", "fetch tweets from a specific user (e.g. elonmusk)")
	bookmarks := flags.String("bookmarks", "", "search your bookmarks (keyword, or 'all' for broad match)")
	browser := flags.String("browser", "firefox", "cookie source: 'firefox' or 'chrome' (Chrome requires macOS)")
	profile := flags.String("profile", "", "browser profile directory name (Chrome: Default or 'Profile 1')")
	queryID := flags.String("query-id", "", "override timeline GraphQL query ID")
	if err := flags.Parse(args); err != nil {
		return err
	}
	// Reject invalid input before opening a browser store or prompting Keychain.
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments; use --help for usage")
	}
	if *browser != "firefox" && *browser != "chrome" {
		return fmt.Errorf("--browser must be 'firefox' or 'chrome'")
	}
	if *count < 1 || *count > 100 {
		return fmt.Errorf("--count must be between 1 and 100")
	}
	if *timeline != "following" && *timeline != "foryou" {
		return fmt.Errorf("--timeline must be 'following' or 'foryou'")
	}
	if *user != "" && *bookmarks != "" {
		return fmt.Errorf("choose either --user or --bookmarks")
	}
	handle := strings.TrimPrefix(*user, "@")
	if *user != "" && handle == "" {
		return fmt.Errorf("--user requires a screen name after @")
	}

	var authToken, ct0 string
	var err error
	switch *browser {
	case "chrome":
		fmt.Fprintln(out, "Reading Google Chrome cookies...")
		authToken, ct0, err = cookies.ExtractChrome(*profile)
	default:
		fmt.Fprintln(out, "Reading Firefox cookies...")
		authToken, ct0, err = cookies.ExtractFirefox(*profile)
	}
	if err != nil {
		return err
	}
	client := twitter.NewClient(authToken, ct0)
	var tweets []twitter.Tweet
	switch {
	case *bookmarks != "":
		q := *bookmarks
		if q == "all" {
			q = ""
		}
		fmt.Fprintln(out, "Searching bookmarks...")
		tweets, err = client.FetchBookmarks(*count, q)
	case *user != "":
		fmt.Fprintf(out, "Fetching tweets from @%s...\n", handle)
		tweets, err = client.FetchUserTweets(handle, *count)
	default:
		fmt.Fprintf(out, "Fetching %s timeline...\n", *timeline)
		tweets, err = client.FetchTimeline(*timeline, *count, *queryID)
	}
	if err != nil {
		return err
	}
	display.TweetsTo(out, tweets, *count)
	return nil
}
