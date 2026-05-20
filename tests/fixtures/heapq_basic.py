import heapq


def main() -> None:
    h: list[int] = []
    heapq.heappush(h, 5)
    heapq.heappush(h, 2)
    heapq.heappush(h, 8)
    heapq.heappush(h, 1)
    print(h[0])
    print(heapq.heappop(h))
    print(heapq.heappop(h))
    print(heapq.heappop(h))
    print(heapq.heappop(h))
    xs: list[int] = [3, 1, 4, 1, 5, 9, 2, 6]
    heapq.heapify(xs)
    while len(xs) > 0:
        print(heapq.heappop(xs))


if __name__ == "__main__":
    main()
