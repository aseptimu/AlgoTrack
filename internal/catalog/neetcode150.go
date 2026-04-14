package catalog

// NeetCode150 is a curated subset of the public NeetCode 150 list, ordered by
// topic in the canonical NeetCode order and by suggested difficulty within
// each topic. Premium-only problems (#271, #261, #323, #269, #252, #253) are
// intentionally excluded so the recommender never points users at locked
// content.
//
// This is not the literal 150-entry list — it covers ~120 of the public,
// non-premium problems. Adding more is a one-line append and does not
// require any code changes elsewhere.
var NeetCode150 = Catalog{
	Name: "NeetCode 150",
	Problems: []Problem{
		// --- Arrays & Hashing ---
		{217, "contains-duplicate", "Contains Duplicate", Easy, "Arrays & Hashing"},
		{242, "valid-anagram", "Valid Anagram", Easy, "Arrays & Hashing"},
		{1, "two-sum", "Two Sum", Easy, "Arrays & Hashing"},
		{49, "group-anagrams", "Group Anagrams", Medium, "Arrays & Hashing"},
		{347, "top-k-frequent-elements", "Top K Frequent Elements", Medium, "Arrays & Hashing"},
		{238, "product-of-array-except-self", "Product of Array Except Self", Medium, "Arrays & Hashing"},
		{36, "valid-sudoku", "Valid Sudoku", Medium, "Arrays & Hashing"},
		{128, "longest-consecutive-sequence", "Longest Consecutive Sequence", Medium, "Arrays & Hashing"},

		// --- Two Pointers ---
		{125, "valid-palindrome", "Valid Palindrome", Easy, "Two Pointers"},
		{167, "two-sum-ii-input-array-is-sorted", "Two Sum II - Input Array Is Sorted", Medium, "Two Pointers"},
		{15, "3sum", "3Sum", Medium, "Two Pointers"},
		{11, "container-with-most-water", "Container With Most Water", Medium, "Two Pointers"},
		{42, "trapping-rain-water", "Trapping Rain Water", Hard, "Two Pointers"},

		// --- Sliding Window ---
		{121, "best-time-to-buy-and-sell-stock", "Best Time to Buy and Sell Stock", Easy, "Sliding Window"},
		{3, "longest-substring-without-repeating-characters", "Longest Substring Without Repeating Characters", Medium, "Sliding Window"},
		{424, "longest-repeating-character-replacement", "Longest Repeating Character Replacement", Medium, "Sliding Window"},
		{567, "permutation-in-string", "Permutation in String", Medium, "Sliding Window"},
		{76, "minimum-window-substring", "Minimum Window Substring", Hard, "Sliding Window"},
		{239, "sliding-window-maximum", "Sliding Window Maximum", Hard, "Sliding Window"},

		// --- Stack ---
		{20, "valid-parentheses", "Valid Parentheses", Easy, "Stack"},
		{155, "min-stack", "Min Stack", Medium, "Stack"},
		{150, "evaluate-reverse-polish-notation", "Evaluate Reverse Polish Notation", Medium, "Stack"},
		{22, "generate-parentheses", "Generate Parentheses", Medium, "Stack"},
		{739, "daily-temperatures", "Daily Temperatures", Medium, "Stack"},
		{853, "car-fleet", "Car Fleet", Medium, "Stack"},
		{84, "largest-rectangle-in-histogram", "Largest Rectangle in Histogram", Hard, "Stack"},

		// --- Binary Search ---
		{704, "binary-search", "Binary Search", Easy, "Binary Search"},
		{74, "search-a-2d-matrix", "Search a 2D Matrix", Medium, "Binary Search"},
		{875, "koko-eating-bananas", "Koko Eating Bananas", Medium, "Binary Search"},
		{153, "find-minimum-in-rotated-sorted-array", "Find Minimum in Rotated Sorted Array", Medium, "Binary Search"},
		{33, "search-in-rotated-sorted-array", "Search in Rotated Sorted Array", Medium, "Binary Search"},
		{981, "time-based-key-value-store", "Time Based Key-Value Store", Medium, "Binary Search"},
		{4, "median-of-two-sorted-arrays", "Median of Two Sorted Arrays", Hard, "Binary Search"},

		// --- Linked List ---
		{206, "reverse-linked-list", "Reverse Linked List", Easy, "Linked List"},
		{21, "merge-two-sorted-lists", "Merge Two Sorted Lists", Easy, "Linked List"},
		{141, "linked-list-cycle", "Linked List Cycle", Easy, "Linked List"},
		{143, "reorder-list", "Reorder List", Medium, "Linked List"},
		{19, "remove-nth-node-from-end-of-list", "Remove Nth Node From End of List", Medium, "Linked List"},
		{138, "copy-list-with-random-pointer", "Copy List with Random Pointer", Medium, "Linked List"},
		{2, "add-two-numbers", "Add Two Numbers", Medium, "Linked List"},
		{287, "find-the-duplicate-number", "Find the Duplicate Number", Medium, "Linked List"},
		{146, "lru-cache", "LRU Cache", Medium, "Linked List"},
		{23, "merge-k-sorted-lists", "Merge k Sorted Lists", Hard, "Linked List"},
		{25, "reverse-nodes-in-k-group", "Reverse Nodes in k-Group", Hard, "Linked List"},

		// --- Trees ---
		{226, "invert-binary-tree", "Invert Binary Tree", Easy, "Trees"},
		{104, "maximum-depth-of-binary-tree", "Maximum Depth of Binary Tree", Easy, "Trees"},
		{543, "diameter-of-binary-tree", "Diameter of Binary Tree", Easy, "Trees"},
		{110, "balanced-binary-tree", "Balanced Binary Tree", Easy, "Trees"},
		{100, "same-tree", "Same Tree", Easy, "Trees"},
		{572, "subtree-of-another-tree", "Subtree of Another Tree", Easy, "Trees"},
		{235, "lowest-common-ancestor-of-a-binary-search-tree", "Lowest Common Ancestor of a Binary Search Tree", Medium, "Trees"},
		{102, "binary-tree-level-order-traversal", "Binary Tree Level Order Traversal", Medium, "Trees"},
		{199, "binary-tree-right-side-view", "Binary Tree Right Side View", Medium, "Trees"},
		{1448, "count-good-nodes-in-binary-tree", "Count Good Nodes in Binary Tree", Medium, "Trees"},
		{98, "validate-binary-search-tree", "Validate Binary Search Tree", Medium, "Trees"},
		{230, "kth-smallest-element-in-a-bst", "Kth Smallest Element in a BST", Medium, "Trees"},
		{105, "construct-binary-tree-from-preorder-and-inorder-traversal", "Construct Binary Tree from Preorder and Inorder Traversal", Medium, "Trees"},
		{124, "binary-tree-maximum-path-sum", "Binary Tree Maximum Path Sum", Hard, "Trees"},
		{297, "serialize-and-deserialize-binary-tree", "Serialize and Deserialize Binary Tree", Hard, "Trees"},

		// --- Tries ---
		{208, "implement-trie-prefix-tree", "Implement Trie (Prefix Tree)", Medium, "Tries"},
		{211, "design-add-and-search-words-data-structure", "Design Add and Search Words Data Structure", Medium, "Tries"},
		{212, "word-search-ii", "Word Search II", Hard, "Tries"},

		// --- Heap / Priority Queue ---
		{703, "kth-largest-element-in-a-stream", "Kth Largest Element in a Stream", Easy, "Heap / Priority Queue"},
		{1046, "last-stone-weight", "Last Stone Weight", Easy, "Heap / Priority Queue"},
		{973, "k-closest-points-to-origin", "K Closest Points to Origin", Medium, "Heap / Priority Queue"},
		{215, "kth-largest-element-in-an-array", "Kth Largest Element in an Array", Medium, "Heap / Priority Queue"},
		{621, "task-scheduler", "Task Scheduler", Medium, "Heap / Priority Queue"},
		{355, "design-twitter", "Design Twitter", Medium, "Heap / Priority Queue"},
		{295, "find-median-from-data-stream", "Find Median from Data Stream", Hard, "Heap / Priority Queue"},

		// --- Backtracking ---
		{78, "subsets", "Subsets", Medium, "Backtracking"},
		{39, "combination-sum", "Combination Sum", Medium, "Backtracking"},
		{46, "permutations", "Permutations", Medium, "Backtracking"},
		{90, "subsets-ii", "Subsets II", Medium, "Backtracking"},
		{40, "combination-sum-ii", "Combination Sum II", Medium, "Backtracking"},
		{79, "word-search", "Word Search", Medium, "Backtracking"},
		{131, "palindrome-partitioning", "Palindrome Partitioning", Medium, "Backtracking"},
		{17, "letter-combinations-of-a-phone-number", "Letter Combinations of a Phone Number", Medium, "Backtracking"},
		{51, "n-queens", "N-Queens", Hard, "Backtracking"},

		// --- Graphs ---
		{200, "number-of-islands", "Number of Islands", Medium, "Graphs"},
		{695, "max-area-of-island", "Max Area of Island", Medium, "Graphs"},
		{133, "clone-graph", "Clone Graph", Medium, "Graphs"},
		{994, "rotting-oranges", "Rotting Oranges", Medium, "Graphs"},
		{417, "pacific-atlantic-water-flow", "Pacific Atlantic Water Flow", Medium, "Graphs"},
		{130, "surrounded-regions", "Surrounded Regions", Medium, "Graphs"},
		{207, "course-schedule", "Course Schedule", Medium, "Graphs"},
		{210, "course-schedule-ii", "Course Schedule II", Medium, "Graphs"},
		{684, "redundant-connection", "Redundant Connection", Medium, "Graphs"},

		// --- Advanced Graphs ---
		{332, "reconstruct-itinerary", "Reconstruct Itinerary", Hard, "Advanced Graphs"},
		{1584, "min-cost-to-connect-all-points", "Min Cost to Connect All Points", Medium, "Advanced Graphs"},
		{743, "network-delay-time", "Network Delay Time", Medium, "Advanced Graphs"},
		{787, "cheapest-flights-within-k-stops", "Cheapest Flights Within K Stops", Medium, "Advanced Graphs"},
		{778, "swim-in-rising-water", "Swim in Rising Water", Hard, "Advanced Graphs"},

		// --- 1-D DP ---
		{70, "climbing-stairs", "Climbing Stairs", Easy, "1-D DP"},
		{746, "min-cost-climbing-stairs", "Min Cost Climbing Stairs", Easy, "1-D DP"},
		{198, "house-robber", "House Robber", Medium, "1-D DP"},
		{213, "house-robber-ii", "House Robber II", Medium, "1-D DP"},
		{5, "longest-palindromic-substring", "Longest Palindromic Substring", Medium, "1-D DP"},
		{647, "palindromic-substrings", "Palindromic Substrings", Medium, "1-D DP"},
		{91, "decode-ways", "Decode Ways", Medium, "1-D DP"},
		{322, "coin-change", "Coin Change", Medium, "1-D DP"},
		{152, "maximum-product-subarray", "Maximum Product Subarray", Medium, "1-D DP"},
		{139, "word-break", "Word Break", Medium, "1-D DP"},
		{300, "longest-increasing-subsequence", "Longest Increasing Subsequence", Medium, "1-D DP"},
		{416, "partition-equal-subset-sum", "Partition Equal Subset Sum", Medium, "1-D DP"},

		// --- 2-D DP ---
		{62, "unique-paths", "Unique Paths", Medium, "2-D DP"},
		{1143, "longest-common-subsequence", "Longest Common Subsequence", Medium, "2-D DP"},
		{309, "best-time-to-buy-and-sell-stock-with-cooldown", "Best Time to Buy and Sell Stock with Cooldown", Medium, "2-D DP"},
		{518, "coin-change-ii", "Coin Change II", Medium, "2-D DP"},
		{494, "target-sum", "Target Sum", Medium, "2-D DP"},
		{97, "interleaving-string", "Interleaving String", Medium, "2-D DP"},
		{72, "edit-distance", "Edit Distance", Medium, "2-D DP"},
		{329, "longest-increasing-path-in-a-matrix", "Longest Increasing Path in a Matrix", Hard, "2-D DP"},
		{115, "distinct-subsequences", "Distinct Subsequences", Hard, "2-D DP"},
		{312, "burst-balloons", "Burst Balloons", Hard, "2-D DP"},
		{10, "regular-expression-matching", "Regular Expression Matching", Hard, "2-D DP"},

		// --- Greedy ---
		{53, "maximum-subarray", "Maximum Subarray", Medium, "Greedy"},
		{55, "jump-game", "Jump Game", Medium, "Greedy"},
		{45, "jump-game-ii", "Jump Game II", Medium, "Greedy"},
		{134, "gas-station", "Gas Station", Medium, "Greedy"},
		{846, "hand-of-straights", "Hand of Straights", Medium, "Greedy"},
		{763, "partition-labels", "Partition Labels", Medium, "Greedy"},
		{678, "valid-parenthesis-string", "Valid Parenthesis String", Medium, "Greedy"},

		// --- Intervals ---
		{57, "insert-interval", "Insert Interval", Medium, "Intervals"},
		{56, "merge-intervals", "Merge Intervals", Medium, "Intervals"},
		{435, "non-overlapping-intervals", "Non-overlapping Intervals", Medium, "Intervals"},
		{1851, "minimum-interval-to-include-each-query", "Minimum Interval to Include Each Query", Hard, "Intervals"},

		// --- Math & Geometry ---
		{48, "rotate-image", "Rotate Image", Medium, "Math & Geometry"},
		{54, "spiral-matrix", "Spiral Matrix", Medium, "Math & Geometry"},
		{73, "set-matrix-zeroes", "Set Matrix Zeroes", Medium, "Math & Geometry"},
		{202, "happy-number", "Happy Number", Easy, "Math & Geometry"},
		{66, "plus-one", "Plus One", Easy, "Math & Geometry"},
		{50, "powx-n", "Pow(x, n)", Medium, "Math & Geometry"},
		{43, "multiply-strings", "Multiply Strings", Medium, "Math & Geometry"},

		// --- Bit Manipulation ---
		{136, "single-number", "Single Number", Easy, "Bit Manipulation"},
		{191, "number-of-1-bits", "Number of 1 Bits", Easy, "Bit Manipulation"},
		{338, "counting-bits", "Counting Bits", Easy, "Bit Manipulation"},
		{190, "reverse-bits", "Reverse Bits", Easy, "Bit Manipulation"},
		{268, "missing-number", "Missing Number", Easy, "Bit Manipulation"},
		{371, "sum-of-two-integers", "Sum of Two Integers", Medium, "Bit Manipulation"},
		{7, "reverse-integer", "Reverse Integer", Medium, "Bit Manipulation"},
	},
}
