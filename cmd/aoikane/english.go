package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/a-kaibu/aoikane/internal/engine"
	"github.com/a-kaibu/aoikane/internal/english"
)

func englishAssetsFlag(flags *flag.FlagSet) *string {
	return flags.String("english-assets", os.Getenv("AOIKANE_ENGLISH_ASSETS"), "English model and voices directory (default: assets/english)")
}

func chooseLanguage(language string, request *engine.Request) error {
	switch strings.ToLower(language) {
	case "":
		return nil
	case "ja":
		if english.VoicePath(request.Voice) == "" {
			return nil
		}
	case "en":
		switch request.Voice {
		case "":
			request.Voice = english.DefaultVoice
		case "KotonohaAoi_ns_48":
			request.Voice = "en-aoi"
		case "KotonohaAkane_ns_48":
			request.Voice = "en-akane"
		}
		if english.VoicePath(request.Voice) != "" {
			return nil
		}
	default:
		return fmt.Errorf("language must be ja or en")
	}
	return fmt.Errorf("voice %q is unavailable for language %s", request.Voice, language)
}

func openForSynthesis(japaneseDir, englishDir, voice string) (engine.Synthesizer, func(), error) {
	if english.VoicePath(voice) == "" {
		return openEngine(japaneseDir)
	}
	if englishDir == "" {
		englishDir = english.DefaultDir
	}
	e, err := english.OpenModel(englishDir)
	if err != nil {
		return nil, nil, err
	}
	if err := e.LoadVoice(englishDir, voice); err != nil {
		e.Close()
		return nil, nil, err
	}
	return e, func() { e.Close() }, nil
}

func openInstalledEngines(japaneseDir, englishDir string) (engine.Synthesizer, func(), error) {
	japanese, closeJapanese, err := openEngine(japaneseDir)
	if err != nil {
		return nil, nil, err
	}
	if englishDir == "" {
		englishDir = english.DefaultDir
		if _, err := os.Stat(filepath.Join(englishDir, "model.safetensors")); os.IsNotExist(err) {
			return japanese, closeJapanese, nil
		}
	}
	englishEngine, err := english.Open(englishDir)
	if err != nil {
		closeJapanese()
		return nil, nil, err
	}
	return installedEngines{japanese, englishEngine}, func() { englishEngine.Close(); closeJapanese() }, nil
}

type installedEngines struct {
	japanese, english engine.Synthesizer
}

func (s installedEngines) Voices() []engine.Voice {
	return append(s.japanese.Voices(), s.english.Voices()...)
}

func (s installedEngines) Synthesize(ctx context.Context, r engine.Request) ([]byte, error) {
	if english.VoicePath(r.Voice) != "" {
		return s.english.Synthesize(ctx, r)
	}
	return s.japanese.Synthesize(ctx, r)
}

func (s installedEngines) Close() error {
	err := s.english.Close()
	if other := s.japanese.Close(); err == nil {
		err = other
	}
	return err
}
