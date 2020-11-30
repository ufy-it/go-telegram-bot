package handlers

func nextStep() stepActionResult {
	return stepActionResult{
		NextAction: Next,
	}
}

func prevStep() stepActionResult {
	return stepActionResult{
		NextAction: Prev,
	}
}

func endConversation() stepActionResult {
	return stepActionResult{
		NextAction: End,
	}
}

func repeatStep() stepActionResult {
	return stepActionResult{
		NextAction: Repeat,
	}
}

func repeatCommand() stepActionResult {
	return stepActionResult{
		NextAction: Begin,
	}
}

func goToCommand(command string) stepActionResult {
	return stepActionResult{
		NextCommandName: command,
		NextAction:      Custom,
	}
}

func closeCommand() stepActionResult {
	return stepActionResult{
		NextAction: Close,
	}
}
