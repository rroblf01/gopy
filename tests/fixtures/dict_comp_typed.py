def main() -> None:
    nums: list[int] = [1, 2, 3, 4, 5]
    squared: dict[int, int] = {n: n * n for n in nums}
    for k in sorted(squared.keys()):
        print(k, squared[k])
    # dict comp with condition
    evens: dict[int, int] = {n: n * 2 for n in nums if n % 2 == 0}
    for k in sorted(evens.keys()):
        print(k, evens[k])


if __name__ == "__main__":
    main()
