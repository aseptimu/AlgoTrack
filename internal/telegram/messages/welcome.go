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

const WelcomeWithProgress = `
Welcome back to AlgoTrack

Progress: %d / %d
Remaining: %d

Keep going 🚀

%s`

const Commands = `Available commands:
/start - show welcome message
/help - show all commands
/add <number> - save solved LeetCode problem
/goal <number> - set or update your goal`

const Help = `
AlgoTrack commands

` + Commands

const GoalUsage = `
Use the command like this:
/goal 300`

const InvalidGoal = `
Please provide a valid positive number.
Example:
/goal 300`
