package registry

import "math/rand"

type WeightedRandomBalancer struct{}

func NewWeightedRandomBalancer() *WeightedRandomBalancer {
	return &WeightedRandomBalancer{}
}

// Select 根据权重进行加权随机选择
// 算法逻辑：
// 1. 计算所有实例的总权重
// 2. 生成 [0, 总权重) 范围内的随机数
// 3. 遍历实例，累加权重，当累计权重 >= 随机数时返回该实例
func (lb *WeightedRandomBalancer) Select(eps []Endpoint) *Endpoint {
	length := len(eps)
	if length == 0 {
		return nil
	}

	// 如果只有一个实例，直接返回
	if length == 1 {
		return &eps[0]
	}

	// 计算总权重
	totalWeight := uint64(0)
	for _, ep := range eps {
		weight := ep.GetWeight()
		if weight == 0 {
			weight = 1 // 权重为 0 的实例默认权重为 1
		}
		totalWeight += uint64(weight)
	}

	// 如果总权重为 0（所有实例权重都为 0），随机返回一个
	if totalWeight == 0 {
		ep := eps[rand.Intn(length)]
		return &ep
	}

	// 生成随机数
	randomWeight := rand.Uint64() % totalWeight

	// 按权重选择实例
	currentWeight := uint64(0)
	for i := range eps {
		weight := eps[i].GetWeight()
		if weight == 0 {
			weight = 1
		}
		currentWeight += uint64(weight)
		if currentWeight > randomWeight {
			return &eps[i]
		}
	}

	// 兰底返回最后一个（理论上不会走到这里）
	return &eps[length-1]
}
