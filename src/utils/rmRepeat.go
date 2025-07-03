package utils

/*
*
去重工具类
*/
func RemoveRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat && arr[i] != "" {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

func RemoveRepeatedElementArr(arr [][]string) [][]string {
	filteredTokenIds := make([][]string, 0)
	seen := make(map[string]bool)

	for _, pair := range arr {
		if len(pair) == 2 {
			key := pair[0] + "," + pair[1]

			if _, exists := seen[key]; !exists {
				filteredTokenIds = append(filteredTokenIds, pair)
				seen[key] = true
			}
		} else if len(pair) == 3 {
			key := pair[0] + "," + pair[1] + "," + pair[2]

			if _, exists := seen[key]; !exists {
				filteredTokenIds = append(filteredTokenIds, pair)
				seen[key] = true
			}
		}
	}
	return filteredTokenIds
}
