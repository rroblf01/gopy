from concurrent.futures import ThreadPoolExecutor


counter = [0]


def work() -> None:
    counter[0] += 1


def main() -> None:
    pool = ThreadPoolExecutor()
    f = pool.submit(work)
    f.result()
    pool.shutdown()
    print(counter[0])


if __name__ == "__main__":
    main()
