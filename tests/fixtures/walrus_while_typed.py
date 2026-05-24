def main() -> None:
    nums = [1, 2, 3, 4, 5]
    i = 0
    total = 0
    while (v := nums[i]) < 4:
        total += v
        i += 1
    print(total, i)
    # numeric accumulator no annotation
    acc = 0
    for n in nums:
        acc += n
    print(acc)
    # float accumulator
    sumf = 0.0
    for n in nums:
        sumf += float(n) * 0.5
    print(sumf)


if __name__ == "__main__":
    main()
