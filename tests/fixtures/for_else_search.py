def find_first(nums: list[int], target: int) -> int:
    for i, n in enumerate(nums):
        if n == target:
            return i
    return -1


def main() -> None:
    nums: list[int] = [10, 20, 30, 40, 50]
    print(find_first(nums, 30))
    print(find_first(nums, 99))
    # for-else
    for n in nums:
        if n == 25:
            print("found 25")
            break
    else:
        print("not found")
    # for-else success path
    for n in nums:
        if n == 200:
            print("found")
            break
    else:
        print("none found")


if __name__ == "__main__":
    main()
