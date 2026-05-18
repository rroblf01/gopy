import random


def main() -> None:
    # Seeding makes the sequence deterministic in CPython. Go's math/rand
    # uses a different PRNG so the actual integers won't match — we just
    # verify that the calls produce values in the expected ranges.
    random.seed(42)
    n: int = random.randint(1, 10)
    print(1 <= n <= 10)
    f: float = random.random()
    print(0.0 <= f < 1.0)


if __name__ == "__main__":
    main()
