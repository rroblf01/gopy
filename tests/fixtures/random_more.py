import random


def main() -> None:
    random.seed(42)
    n = random.randrange(0, 100)
    print(0 <= n and n < 100)
    bits: int = random.getrandbits(8)
    print(0 <= bits and bits < 256)
    v = random.gauss(0.0, 1.0)
    print(-100.0 < v and v < 100.0)
    e = random.expovariate(1.0)
    print(e >= 0.0)


if __name__ == "__main__":
    main()
