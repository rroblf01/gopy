import time


def main() -> None:
    t = time.strptime("2024-05-26 10:11:12", "%Y-%m-%d %H:%M:%S")
    print(t[0])
    print(t[1])
    print(t[2])
    print(t[3])


if __name__ == "__main__":
    main()
