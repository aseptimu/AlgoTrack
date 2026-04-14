package catalog

// LeetCodeJS30 is a curated subset of LeetCode's "30 Days of JavaScript"
// study plan. The problem numbers live in the 2600–2900 range and are
// JS-specific. Recommended only when the user opts in via /mode js.
//
// Difficulty labels follow LeetCode's own. Problems that don't have a
// reliable Easy/Medium label here are marked Easy as a safe default.
var LeetCodeJS30 = Catalog{
	Name: "LeetCode 30 Days of JavaScript",
	Problems: []Problem{
		// Closures
		{2667, "create-hello-world-function", "Create Hello World Function", Easy, "JS Closures"},
		{2620, "counter", "Counter", Easy, "JS Closures"},
		{2665, "counter-ii", "Counter II", Easy, "JS Closures"},
		// Basic Array Transformations
		{2635, "apply-transform-over-each-element-in-array", "Apply Transform Over Each Element in Array", Easy, "JS Array Transforms"},
		{2634, "filter-elements-from-array", "Filter Elements from Array", Easy, "JS Array Transforms"},
		{2626, "array-reduce-transformation", "Array Reduce Transformation", Easy, "JS Array Transforms"},
		// Function Transformations
		{2629, "function-composition", "Function Composition", Easy, "JS Function Transforms"},
		{2666, "allow-one-function-call", "Allow One Function Call", Easy, "JS Function Transforms"},
		{2623, "memoize", "Memoize", Medium, "JS Function Transforms"},
		{2625, "flatten-deeply-nested-array", "Flatten Deeply Nested Array", Medium, "JS Function Transforms"},
		{2624, "snail-traversal", "Snail Traversal", Medium, "JS Function Transforms"},
		// Promises and Time
		{2723, "add-two-promises", "Add Two Promises", Easy, "JS Promises"},
		{2725, "interval-cancellation", "Interval Cancellation", Easy, "JS Promises"},
		{2637, "promise-time-limit", "Promise Time Limit", Medium, "JS Promises"},
		{2622, "cache-with-time-limit", "Cache With Time Limit", Medium, "JS Promises"},
		{2627, "debounce", "Debounce", Medium, "JS Promises"},
		{2721, "execute-asynchronous-functions-in-parallel", "Execute Asynchronous Functions in Parallel", Medium, "JS Promises"},
		// JSON
		{2727, "is-object-empty", "Is Object Empty", Easy, "JS JSON"},
		{2677, "chunk-array", "Chunk Array", Easy, "JS JSON"},
		{2619, "array-prototype-last", "Array Prototype Last", Easy, "JS JSON"},
		{2631, "group-by", "Group By", Medium, "JS JSON"},
		{2628, "json-deep-equal", "JSON Deep Equal", Medium, "JS JSON"},
		{2633, "convert-object-to-json-string", "Convert Object to JSON String", Medium, "JS JSON"},
		// Classes
		{2694, "event-emitter", "Event Emitter", Medium, "JS Classes"},
		{2695, "array-wrapper", "Array Wrapper", Easy, "JS Classes"},
		{2622, "calculator-with-method-chaining", "Calculator with Method Chaining", Easy, "JS Classes"},
		{2630, "memoize-ii", "Memoize II", Hard, "JS Classes"},
		// Currying / advanced
		{2632, "curry", "Curry", Medium, "JS Curry"},
		{2693, "call-function-with-custom-context", "Call Function with Custom Context", Medium, "JS Curry"},
		{2675, "array-of-objects-to-matrix", "Array of Objects to Matrix", Hard, "JS Curry"},
	},
}
