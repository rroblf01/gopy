import threading


counter = [0]


def worker() -> None:
    counter[0] += 1


def main() -> None:
    t = threading.Thread(target=worker)
    t.start()
    t.join()
    print(counter[0])


if __name__ == "__main__":
    main()
