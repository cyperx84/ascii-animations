package main

import "github.com/cyperx84/ascii-animations/asciifx/fx"

// banner and bannerFonts wrap the engine's banner renderer so the CLI has a
// single seam for it.
func banner(text, font string) (string, error) { return fx.Banner(text, font) }

func bannerFonts() []string { return fx.BannerFonts() }
