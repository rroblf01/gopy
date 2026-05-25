import random


def main() -> None:
    random.seed(42)
    pool: list[int] = [1, 2, 3, 4, 5]
    picks: list[int] = random.choices(pool, k=10)
    # CPython and gopy use different PRNGs, so don't compare values.
    # Verify length, every pick is from the pool, and the result is
    # non-empty.
    print(len(picks))
    for p in picks:
        if p not in pool:
            print("out-of-pool", p)
            return
    print("all-in-pool")


if __name__ == "__main__":
    main()
