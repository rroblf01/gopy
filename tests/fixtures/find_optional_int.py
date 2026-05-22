from typing import Optional


def find_index(xs: list[int], target: int) -> Optional[int]:
    for i, x in enumerate(xs):
        if x == target:
            return i
    return None


def main() -> None:
    nums: list[int] = [10, 20, 30, 40]
    r1 = find_index(nums, 30)
    print(r1)
    r2 = find_index(nums, 99)
    print(r2)
    if r1 is not None:
        print("found at", r1)


if __name__ == "__main__":
    main()
