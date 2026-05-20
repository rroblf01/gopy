import heapq


def main() -> None:
    xs: list[int] = [5, 1, 4, 1, 5, 9, 2, 6, 5, 3]
    for v in heapq.nsmallest(3, xs):
        print(v)
    print("---")
    for v in heapq.nlargest(3, xs):
        print(v)
    print("---")
    ys: list[str] = ["banana", "apple", "cherry", "date"]
    for v in heapq.nsmallest(2, ys):
        print(v)


if __name__ == "__main__":
    main()
