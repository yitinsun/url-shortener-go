package web

import "testing"

func TestPlainURLIsSafeAndMarkupIsUnsafe(t *testing.T) {
	safe := []string{
		"https://example.com/system-design",
		"https://example.com/path",
		"https://example.com/search?q=rust&lang=en",
		"https://www.google.com/search?q=weather+today&sca_esv=d692c529a49cacda&sxsrf=AKL5yDgYdS29-XsLWACkqg-AdaNBYoCJiw%3A1794927563865&source=hp&ei=y2v8asSAMoutmtkPge7oyQ8&iflsig=AHSINFcAAAAAavx522ARvCQUIIQe3RX5ck_E1MTJpLFP&oq=weath&gs_lp=Egdnd3Mtd2l6IgV3ZWF0aCoQCAIYgAQYigUYQxixAxiDATITEAAYgAQYigUYQxixAxiDARjJAzINEAAYgAQYigUYQxixAzIQEAAYgAQYigUYQxixAxiDATIIEAAYgAQYkgMyDhAAGIAEGLEDGIMBGJIDMhAQABiABBiKBRhDGLEDGIMBMhMQLhiABBiKBRhDGLEDGMcBGNEDMgsQABiABBixAxiDATILEAAYgAQYsQMYgwEyChAAGIAEGIoFGENIrnBQ7ydYmUVwAXgAkAEAmAE4oAGKAqoBATW4AQHIAQD4AQGYAgagAswCqAIKwgIHECMY6gIYJ8ICBBAjGCfCAhAQLhiABBiKBRhDGLEDGIMBwgIREC4YgAQYsQMYgwEYxwEY0QPCAhYQLhiABBiKBRhDGLEDGIMBGMcBGNEDwgINEAAYgAQYigUYQxi0B5gDEfEFXxhT_gOYCT2SBwE2oAfQLLIHATW4B7sCwgcFMi0zLjPIBzCACAE&sclient=gws-wiz",
		"",
		"   ",
	}
	for _, value := range safe {
		if LooksUnsafe(value) {
			t.Fatalf("safe value rejected: %q", value)
		}
	}
	unsafe := []string{
		"https://example.com/<script>alert(1)</script>",
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(1)>",
		"<b>bold</b>",
	}
	for _, value := range unsafe {
		if !LooksUnsafe(value) {
			t.Fatalf("markup accepted: %q", value)
		}
	}
}
