module github.com/a-kaibu/aoikane

go 1.26

replace github.com/braheezy/shine-mp3 => ./third_party/shine-mp3

require (
	github.com/braheezy/shine-mp3 v0.2.0
	github.com/go-audio/audio v1.0.0
	github.com/go-audio/wav v1.1.0
)

require github.com/go-audio/riff v1.0.0 // indirect
