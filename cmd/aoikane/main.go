package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/a-kaibu/aoikane/internal/audio"
	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/httpapi"
)

var version = "0.1.0"

const usage = `aoikane - text-to-speech CLI / HTTP API

  aoikane --text "こんにちは" --output speech.wav
  aoikane synth --text "こんにちは" --output speech.mp3
  aoikane convert --input speech.wav --output speech.mp3
  aoikane script.txt --voice aoi --srt --vtt
  aoikane script.txt --voice akane --srt --vtt
  aoikane version
  aoikane help
  aoikane serve --listen 127.0.0.1:8080
  aoikane voices
  aoikane synth --url http://SERVER_HOST:8080 --text "こんにちは" -o speech.wav

The default command is synth; no input reads UTF-8 stdin.
Use synth --help for synthesis options. Text files and stdin must be UTF-8.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		fmt.Fprintln(os.Stderr, "aoikane:", err)
		os.Exit(1)
	}
}
func flags(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ContinueOnError) }
func run(args []string) error {
	if len(args) == 0 {
		return synth(nil)
	}
	if args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Print(usage + backendUsage)
		return nil
	}
	switch args[0] {
	case "version", "--version", "-version", "-v":
		fmt.Printf("aoikane %s %s/%s\n", version, runtime.GOOS, runtime.GOARCH)
		return nil
	case "pack", "bundle":
		return nativeCommand(args)
	case "convert":
		return convert(args[1:])
	case "synth":
		return synth(args[1:])
	case "voices":
		return voices(args[1:])
	case "serve":
		return serve(args[1:])
	default:
		return synth(args)
	}
}
func synth(args []string) error {
	f := flags("synth")
	text := f.String("text", "", "text to synthesize")
	file := f.String("file", "", "UTF-8 input file; - for stdin (also accepts a positional filename)")
	f.StringVar(file, "input", "", "alias for --file")
	out := f.String("output", "", "audio path; default: input name with .wav/.mp3; - for stdout")
	f.StringVar(out, "o", "", "alias for --output")
	srt := f.String("srt", "", "write SRT subtitles; optional path (default: audio/input name + .srt)")
	vtt := f.String("vtt", "", "write WebVTT subtitles; optional path (default: audio/input name + .vtt)")
	dir := engineFlags(f)
	address := f.String("url", os.Getenv("AOIKANE_URL"), "remote HTTP API")
	token := f.String("token", os.Getenv("AOIKANE_TOKEN"), "API bearer token")
	r := engine.Request{}
	f.StringVar(&r.Format, "format", "", "wav or mp3 (default: inferred from output extension)")
	f.Func("voice", "aoi, akane, or a voice ID (default: first installed voice)", func(value string) error {
		switch strings.ToLower(value) {
		case "aoi", "葵":
			r.Voice = "KotonohaAoi_ns_48"
		case "akane", "茜":
			r.Voice = "KotonohaAkane_ns_48"
		default:
			r.Voice = value
		}
		return nil
	})
	f.Float64Var(&r.Rate, "rate", 1, "speech rate")
	f.Float64Var(&r.Pitch, "pitch", 1, "pitch")
	f.Float64Var(&r.Volume, "volume", 1, "volume")
	f.Float64Var(&r.Emphasis, "emphasis", 1, "intonation emphasis")
	if err := f.Parse(normalizeSynthArgs(f, args)); err != nil {
		return err
	}
	if f.NArg() > 1 {
		return errors.New("use one input filename, --text, or --file")
	}
	if f.NArg() == 1 {
		if *file != "" || *text != "" {
			return errors.New("choose a positional filename, --text, or --file")
		}
		*file = f.Arg(0)
	}
	srtEnabled, vttEnabled := false, false
	f.Visit(func(v *flag.Flag) {
		if v.Name == "srt" {
			srtEnabled = true
		}
		if v.Name == "vtt" {
			vttEnabled = true
		}
	})
	if r.Format == "" {
		r.Format = "wav"
		if strings.EqualFold(filepath.Ext(*out), ".mp3") {
			r.Format = "mp3"
		}
	}
	base := "speech"
	if *file != "" && *file != "-" {
		base = withoutExtension(*file)
	}
	if *out == "" {
		*out = base + "." + r.Format
	}
	if *out != "-" {
		base = withoutExtension(*out)
	}
	if srtEnabled && *srt == "" {
		*srt = base + ".srt"
	}
	if vttEnabled && *vtt == "" {
		*vtt = base + ".vtt"
	}
	if *text == "" && *file == "" {
		*file = "-"
	}
	if *text != "" && *file != "" {
		return errors.New("choose --text or --file")
	}
	r.Text = *text
	if *file != "" {
		var reader io.Reader = os.Stdin
		var input *os.File
		if *file != "-" {
			var err error
			input, err = os.Open(*file)
			if err != nil {
				return err
			}
			defer input.Close()
			reader = input
		}
		b, err := io.ReadAll(io.LimitReader(reader, 65537))
		if err != nil {
			return err
		}
		r.Text = string(b)
	}
	if err := r.Validate(); err != nil {
		return err
	}
	inputPath := *file
	if inputPath == "-" {
		inputPath = ""
	}
	if err := validateOutputPaths(*out, *srt, *vtt, inputPath); err != nil {
		return err
	}
	subtitles := *srt != "" || *vtt != ""
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	var cues []engine.Cue
	var data []byte
	var err error
	if *address != "" {
		c, e := httpapi.NewClient(*address, *token)
		if e != nil {
			return e
		}
		request := r
		if subtitles {
			request.Format = "wav"
		}
		data, err = c.Synthesize(ctx, request)
		if err == nil && subtitles {
			cues, err = audio.WholeAudioCue(data, r.Text)
			if err == nil {
				data, err = audio.Convert(data, r.Format)
			}
		}
	} else {
		s, cleanup, e := openEngine(*dir)
		if e != nil {
			return e
		}
		defer cleanup()
		if timed, ok := s.(engine.TimedSynthesizer); subtitles && ok {
			data, cues, err = timed.SynthesizeTimed(ctx, r)
		} else {
			data, err = s.Synthesize(ctx, r)
			if err == nil && subtitles {
				cues, err = audio.WholeAudioCue(data, r.Text)
			}
		}
		if err == nil {
			data, err = audio.Convert(data, r.Format)
		}
	}
	if err != nil {
		return err
	}
	type sidecar struct {
		path string
		data []byte
	}
	var files []sidecar
	for _, spec := range []struct{ path, format string }{{*srt, "srt"}, {*vtt, "vtt"}} {
		if spec.path == "" {
			continue
		}
		b, e := audio.Subtitles(cues, spec.format)
		if e != nil {
			return e
		}
		files = append(files, sidecar{spec.path, b})
	}
	if *out == "-" {
		if _, err = os.Stdout.Write(data); err != nil {
			return err
		}
	} else {
		if err = os.WriteFile(*out, data, 0600); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Saved %s (%d bytes)\n", *out, len(data))
	}
	for _, file := range files {
		if err = os.WriteFile(file.path, file.data, 0600); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Saved %s (%d bytes)\n", file.path, len(file.data))
	}
	return nil
}

func convert(args []string) error {
	f := flags("convert")
	in := f.String("input", "", "input 16-bit PCM WAV")
	out := f.String("output", "speech.mp3", "output MP3; - for stdout")
	f.StringVar(out, "o", "speech.mp3", "output MP3")
	if err := f.Parse(args); err != nil {
		return err
	}
	if *in == "" || f.NArg() != 0 {
		return errors.New("use --input WAV --output MP3")
	}
	data, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	if len(data) > 256<<20 {
		return errors.New("input exceeds 256 MiB")
	}
	data, err = audio.Convert(data, "mp3")
	if err != nil {
		return err
	}
	if *out == "-" {
		_, err = os.Stdout.Write(data)
		return err
	}
	if err = os.WriteFile(*out, data, 0600); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Saved %s (%d bytes)\n", *out, len(data))
	return nil
}
func voices(args []string) error {
	f := flags("voices")
	dir := engineFlags(f)
	address := f.String("url", os.Getenv("AOIKANE_URL"), "remote HTTP API")
	token := f.String("token", os.Getenv("AOIKANE_TOKEN"), "API bearer token")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected voices arguments")
	}
	var v []engine.Voice
	if *address != "" {
		c, err := httpapi.NewClient(*address, *token)
		if err != nil {
			return err
		}
		v, err = c.ListVoices(context.Background())
		if err != nil {
			return err
		}
	} else {
		s, cleanup, err := openEngine(*dir)
		if err != nil {
			return err
		}
		defer cleanup()
		v = s.Voices()
	}
	return json.NewEncoder(os.Stdout).Encode(v)
}
func serve(args []string) error {
	f := flags("serve")
	listen := f.String("listen", "127.0.0.1:8080", "HTTP listen address")
	dir := engineFlags(f)
	token := f.String("token", os.Getenv("AOIKANE_TOKEN"), "optional API bearer token")
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected serve arguments")
	}
	listener, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	defer listener.Close()
	s, cleanup, err := openEngine(*dir)
	if err != nil {
		return err
	}
	defer cleanup()
	server := &http.Server{Handler: httpapi.Handler(s, *token), ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	server.BaseContext = func(net.Listener) context.Context { return ctx }
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		server.Shutdown(shutdown)
	}()
	fmt.Fprintf(os.Stderr, "Aoikane ready: http://%s (%d voice(s))\n", listener.Addr(), len(s.Voices()))
	err = server.Serve(listener)
	cancel()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Reject aliases before synthesis so subtitles cannot overwrite audio.
func validateOutputPaths(paths ...string) error {
	seen := map[string]bool{}
	for i, path := range paths {
		if path == "" {
			continue
		}
		if path == "-" {
			if i != 0 {
				return errors.New("subtitle output must be a file path")
			}
			continue
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if runtime.GOOS == "windows" {
			abs = strings.ToLower(abs)
		}
		if seen[abs] {
			return errors.New("audio and subtitle output paths must be distinct")
		}
		seen[abs] = true
	}
	return nil
}

func withoutExtension(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

// Go's flag parser stops at positional arguments. Keep option values together
// while allowing a filename before or after the options, and optional sidecar paths.
func normalizeSynthArgs(f *flag.FlagSet, args []string) []string {
	var options, positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			continue
		}
		name, _, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name == "srt" || name == "vtt" {
			if !hasValue {
				arg += "="
				if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
					i++
					arg += args[i]
				}
			}
			options = append(options, arg)
			continue
		}
		options = append(options, arg)
		if !hasValue && f.Lookup(name) != nil && i+1 < len(args) {
			i++
			options = append(options, args[i])
		}
	}
	return append(append(options, "--"), positional...)
}
