import time


def main() -> None:
    a = time.monotonic()
    time.sleep(0.01)
    b = time.monotonic()
    print(b > a)
    c = time.perf_counter()
    print(c > 0)
    n = time.time_ns()
    print(n > 0)


if __name__ == "__main__":
    main()
