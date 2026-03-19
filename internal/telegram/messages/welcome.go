package messages

const WelcomeNoGoal = `
Welcome to AlgoTrack

This bot helps you:
• Track solved algorithm problems
• Repeat them with spaced repetition
• Reach your goal (e.g. 700 problems)

Let's start.

What is your goal?

%s`

const Commands = `Available commands:
/start - show welcome message
/help - show all commands
/add &lt;number&gt; - save solved LeetCode problem or mark repetition
/goal &lt;number&gt; [easy|medium|hard] - set or update your goal`

const Help = `
AlgoTrack commands

` + Commands

const GoalUsage = `
Use the command like this:
/goal 300
/goal 100 easy
/goal 300 medium`

const InvalidGoal = `
Please provide a valid positive number.
Example:
/goal 300
/goal 100 easy
/goal 300 medium`

const InvalidGoalDifficulty = `
Use one of the supported difficulty levels: easy, medium, hard.
Examples:
/goal 100 easy
/goal 300 medium`

const GoalSavedNoProgress = `
<b>Goal saved</b>.`

const GoalSavedWithProgress = `
<b>Goal saved</b>.

<b>Current progress</b>:
%s`

func WelcomeWithProgress(progressLines string) string {
	return `
<b>Welcome back to AlgoTrack</b>

<b>Goals</b>:
` + progressLines + `

Keep going 🚀

%s`
}
