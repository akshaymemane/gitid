Output docs/public/demo.gif

Require go

Set Shell "bash"
Set Theme "Catppuccin Mocha"
Set FontSize 24
Set Width 1200
Set Height 760
Set Padding 22
Set WindowBar Colorful
Set TypingSpeed 60ms

Hide
Type "DEMO_DIR=/tmp/gitid-demo; rm -rf $DEMO_DIR; mkdir -p $DEMO_DIR/bin $DEMO_DIR/home $DEMO_DIR/acme-api" Enter
Type "go build -o $DEMO_DIR/bin/gitid ./cmd/gitid" Enter
Type "export PATH=$DEMO_DIR/bin:$PATH" Enter
Type "export HOME=$DEMO_DIR/home GITID_CONFIG_DIR=$DEMO_DIR/config FORCE_COLOR=1 TERM=xterm-256color" Enter
Type "cd $DEMO_DIR/acme-api && git init -q" Enter
Type "gitid setup >/dev/null" Enter
Type "clear" Enter
Show

Type "gitid add" Enter
Sleep 3s
Type "work" Enter
Sleep 700ms
Type "Akshay Memane" Enter
Sleep 700ms
Type "akshay@acme.dev" Enter
Sleep 700ms
Type "akshay-work" Enter
Sleep 700ms
Enter
Type "n"
Sleep 600ms
Type "n"
Sleep 3s

Type "gitid add oss --user 'Akshay Memane' --email 'akshay@oss.dev' --github akshay-oss" Enter
Sleep 3s

Hide
Type "clear" Enter
Show
Type "gitid list" Enter
Sleep 4s

Type "gitid switch work" Enter
Sleep 3s

Hide
Type "clear" Enter
Show
Type "gitid list" Enter
Sleep 4s

Type "gitid switch oss" Enter
Sleep 3s

Hide
Type "clear" Enter
Show
Type "gitid list" Enter
Sleep 5s
