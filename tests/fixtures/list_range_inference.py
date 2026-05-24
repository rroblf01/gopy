from itertools import islice


def main() -> None:
    nums = list(range(20))
    print(list(islice(nums, 5)))
    print(list(islice(nums, 5, 10)))
    print(list(islice(nums, 0, 20, 3)))
    print(sorted(nums))
    # min/max on inferred list
    print(min(nums))
    print(max(nums))
    # sum
    print(sum(nums))
    # length
    print(len(nums))


if __name__ == "__main__":
    main()
