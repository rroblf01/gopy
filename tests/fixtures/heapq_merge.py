import heapq


def main() -> None:
    a: list[int] = [1, 3, 5, 7]
    b: list[int] = [2, 4, 6, 8]
    merged = list(heapq.merge(a, b))
    for v in merged:
        print(v)


if __name__ == "__main__":
    main()
