import sched


def main() -> None:
    s = sched.scheduler()
    print(s.empty())
    s.run()
    print("done")


if __name__ == "__main__":
    main()
