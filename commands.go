package onerng

const (
	// CmdVersion - print firmware version (as "Version n")
	CmdVersion = "cmdv\n"
	// CmdFlush - flush entropy pool
	CmdFlush = "cmdw\n"
	// CmdImage - extract the signed firmware image for verification
	CmdImage = "cmdX\n"
	// CmdID - print hardware ID
	CmdID = "cmdI\n"
	// CmdRun - start the task
	CmdRun = "cmdO\n"
	// CmdPause - stop/pause the task
	CmdPause = "cmdo\n"

	// CmdAvalancheWhitener - Avalanche noise with whitener (default)
	CmdAvalancheWhitener = "cmd0\n"
	// CmdAvalanche - Raw avalanche noise
	CmdAvalanche = "cmd1\n"
	// CmdAvalancheRFWhitener - Avalanche noise and RF noise with whitener
	CmdAvalancheRFWhitener = "cmd2\n"
	// CmdAvalancheRF - Raw avalanche noise and RF noise
	CmdAvalancheRF = "cmd3\n"
	// CmdSilent - No noise (necessary for image extraction)
	CmdSilent = "cmd4\n"
	// CmdSilent2 - No noise
	CmdSilent2 = "cmd5\n"
	// CmdRFWhitener - RF noise with whitener
	CmdRFWhitener = "cmd6\n"
	// CmdRF - Raw RF noise
	CmdRF = "cmd7\n"
)

const (
	// DisableWhitener - Disable the on-board CRC16 generator - no effect if both noise generators are disabled
	DisableWhitener ReadMode = 1 << iota
	// EnableRF - Enable noise generation from RF
	EnableRF
	// DisableAvalanche - Disable noise generation from the Avalanche Diode
	DisableAvalanche

	// Default - Avalanche enabled, RF disabled, Whitener enabled.
	Default ReadMode = 0
	// Silent - a convenience - everything disabled
	Silent ReadMode = 4
)
