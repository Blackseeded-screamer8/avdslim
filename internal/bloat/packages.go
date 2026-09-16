package bloat

// StandardBloatCategories lists non-essential background packages safe to disable
// for everyday Android / Flutter development.
// Core OS functionality, WebView, Flutter runtime, networking, and Google Play
// Services core APIs (Auth, FCM push notifications) remain fully functional.
var StandardBloatCategories = map[string][]string{
	"Assistant & Search (~250-400 MB RAM)": {
		"com.google.android.googlequicksearchbox",
		"com.google.android.as",
		"com.google.android.as.oss",
	},
	"Media & Consumer Apps (~150-250 MB RAM)": {
		"com.google.android.apps.photos",
		"com.google.android.youtube",
		"com.google.android.videos",
		"com.google.android.music",
		"com.google.android.apps.youtube.music",
		"com.google.android.apps.maps",
		"com.google.android.gm",
		"com.android.gallery3d",
		"com.android.music",
	},
	"Telephony & Messaging Daemons (~100-150 MB RAM)": {
		"com.google.android.apps.messaging",
		"com.google.android.dialer",
		"com.google.android.contacts",
		"com.android.mms",
		"com.android.dialer",
		"com.android.contacts",
	},
	"Accessibility & Speech (~50-100 MB RAM)": {
		"com.google.android.tts",
		"com.google.android.marvin.talkback",
	},
	"Printing & Background Spoolers (~30-60 MB RAM)": {
		"com.android.printspooler",
		"com.android.bips",
		"com.google.android.printservice.recommendation",
	},
	"Telemetry, Feedback & Screensavers (~40-80 MB RAM)": {
		"com.google.android.feedback",
		"com.android.traceur",
		"com.android.dreams.basic",
		"com.android.dreams.phototable",
		"com.android.wallpaper.livepicker",
		"com.google.android.apps.wallpaper",
		"com.android.emergency",
		"com.android.calendar",
		"com.google.android.calendar",
		"com.android.deskclock",
		"com.google.android.deskclock",
		"com.android.calculator2",
		"com.google.android.calculator",
	},
}

// AggressiveBloatPackages includes packages that can be disabled when maximum
// resource conservation is desired.
var AggressiveBloatPackages = []string{
	"com.android.vending", // Google Play Store background auto-updater
	"com.android.chrome",  // Chrome background sync (Webview remains functional)
	"com.google.android.partnersetup",
	"com.google.android.setupwizard",
	"com.android.setupwizard",
}
