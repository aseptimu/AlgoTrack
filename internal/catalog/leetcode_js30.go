package catalog

// LeetCodeJS30 mirrors LeetCode's "30 Days of JavaScript" study plan,
// preserving its section order. Topic labels follow the plan's own section
// headers; the five problems that follow "Classes" on the page are LeetCode's
// own Bonus Challenges, tagged "JS Bonus" here.
var LeetCodeJS30 = Catalog{
	Name: "LeetCode 30 Days of JavaScript",
	Problems: []Problem{
		// Closures
		{2667, "create-hello-world-function", "Create Hello World Function", Easy, "JS Closures"},
		{2620, "counter", "Counter", Easy, "JS Closures"},
		{2704, "to-be-or-not-to-be", "To Be Or Not To Be", Easy, "JS Closures"},
		{2665, "counter-ii", "Counter II", Easy, "JS Closures"},
		// Basic Array Transformations
		{2635, "apply-transform-over-each-element-in-array", "Apply Transform Over Each Element in Array", Easy, "JS Basic Array Transformations"},
		{2634, "filter-elements-from-array", "Filter Elements from Array", Easy, "JS Basic Array Transformations"},
		{2626, "array-reduce-transformation", "Array Reduce Transformation", Easy, "JS Basic Array Transformations"},
		// Function Transformations
		{2629, "function-composition", "Function Composition", Easy, "JS Function Transformations"},
		{2703, "return-length-of-arguments-passed", "Return Length of Arguments Passed", Easy, "JS Function Transformations"},
		{2666, "allow-one-function-call", "Allow One Function Call", Easy, "JS Function Transformations"},
		{2623, "memoize", "Memoize", Medium, "JS Function Transformations"},
		// Promises and Time
		{2723, "add-two-promises", "Add Two Promises", Easy, "JS Promises and Time"},
		{2621, "sleep", "Sleep", Easy, "JS Promises and Time"},
		{2715, "timeout-cancellation", "Timeout Cancellation", Easy, "JS Promises and Time"},
		{2725, "interval-cancellation", "Interval Cancellation", Easy, "JS Promises and Time"},
		{2637, "promise-time-limit", "Promise Time Limit", Medium, "JS Promises and Time"},
		{2622, "cache-with-time-limit", "Cache With Time Limit", Medium, "JS Promises and Time"},
		{2627, "debounce", "Debounce", Medium, "JS Promises and Time"},
		{2721, "execute-asynchronous-functions-in-parallel", "Execute Asynchronous Functions in Parallel", Medium, "JS Promises and Time"},
		// JSON
		{2727, "is-object-empty", "Is Object Empty", Easy, "JS JSON"},
		{2677, "chunk-array", "Chunk Array", Easy, "JS JSON"},
		{2619, "array-prototype-last", "Array Prototype Last", Easy, "JS JSON"},
		{2631, "group-by", "Group By", Medium, "JS JSON"},
		{2724, "sort-by", "Sort By", Easy, "JS JSON"},
		{2722, "join-two-arrays-by-id", "Join Two Arrays by ID", Medium, "JS JSON"},
		{2625, "flatten-deeply-nested-array", "Flatten Deeply Nested Array", Medium, "JS JSON"},
		{2705, "compact-object", "Compact Object", Medium, "JS JSON"},
		// Classes
		{2694, "event-emitter", "Event Emitter", Medium, "JS Classes"},
		{2695, "array-wrapper", "Array Wrapper", Easy, "JS Classes"},
		{2726, "calculator-with-method-chaining", "Calculator with Method Chaining", Easy, "JS Classes"},
		// Bonus Challenges
		{2636, "promise-pool", "Promise Pool", Medium, "JS Bonus"},
		{2676, "throttle", "Throttle", Medium, "JS Bonus"},
		{2632, "curry", "Curry", Medium, "JS Bonus"},
		{2628, "json-deep-equal", "JSON Deep Equal", Medium, "JS Bonus"},
		{2633, "convert-object-to-json-string", "Convert Object to JSON String", Medium, "JS Bonus"},
	},
}
