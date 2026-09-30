# Run from the repository root after make build. Requires VHS 0.10+, ttyd, ffmpeg.
# Live X requests using the local Chrome login. Output changes with your account and time.
# Review the recording before sharing: it includes your feeds and bookmark search results.
Output docs/demo.mp4
Output docs/demo.gif
Output docs/demo.ascii

Set Shell bash
Set FontFamily "Menlo"
Set FontSize 22
Set Width 1440
Set Height 1080
Set Padding 28
Set Margin 20
Set MarginFill "#101522"
Set BorderRadius 12
Set WindowBar Colorful
Set Theme "Catppuccin Mocha"
Set TypingSpeed 22ms
Set Framerate 20
Env PS1 "❯ "
Env TERM "xterm-256color"
Env COLORTERM "truecolor"
Env CLICOLOR_FORCE "1"
Env BASH_SILENCE_DEPRECATION_WARNING "1"
Env HISTFILE "/dev/null"

Hide
Type "export PS1='❯ '; clear"
Enter
Sleep 500ms
Show
Type "# X-DIGEST / Live X results from a local Chrome session"
Enter
Type "./x-digest --browser chrome --profile Default --count 2"
Sleep 400ms
Enter
Wait+Line@65s /❯\s*$/
Sleep 6s

Hide
Type "clear"
Enter
Sleep 200ms
Show
Type "# 02 / For You: a different feed, ranked by engagement"
Enter
Type "./x-digest --browser chrome --timeline foryou --count 2"
Sleep 400ms
Enter
Wait+Line@65s /❯\s*$/
Sleep 6s

Hide
Type "clear"
Enter
Sleep 200ms
Show
Type "# 03 / Account lookup: @ prefix accepted"
Enter
Type "./x-digest --browser chrome --user @karpathy --count 2"
Sleep 400ms
Enter
Wait+Line@65s /❯\s*$/
Sleep 6s

Hide
Type "clear"
Enter
Sleep 200ms
Show
Type "# 04 / Search bookmarked posts by keyword"
Enter
Type './x-digest --browser chrome --bookmarks "claude" --count 1'
Sleep 400ms
Enter
Wait+Line@65s /❯\s*$/
Sleep 6s

Hide
Type "clear"
Enter
Sleep 200ms
Show
Type "# 05 / Empty results are handled cleanly"
Enter
Type './x-digest --browser chrome --bookmarks "x-digest-no-matching-bookmark-20260930"'
Sleep 400ms
Enter
Wait+Line@65s /❯\s*$/
Sleep 3s
Type "# Live requests complete. Cookie values are never displayed."
Enter
Sleep 3s
