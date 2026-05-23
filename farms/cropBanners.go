package farms

import "strings"

func getCropBanner(cropname string) string {
	m := map[string]string{
		// fruits
		"apple":         "https://i.pinimg.com/736x/a5/48/3a/a5483aa1fd2a077d4418f402c4200582.jpg",
		"banana":        "https://i.pinimg.com/1200x/60/6c/3a/606c3a5d63267b1272b9d03bd44698fb.jpg",
		"mango":         "https://i.pinimg.com/736x/28/90/61/289061a99c88af30b131daa45e4d775d.jpg",
		"orange":        "https://i.pinimg.com/736x/6b/35/19/6b3519ebd502c07520b442783f345d7a.jpg",
		"grapes":        "https://i.pinimg.com/1200x/2e/34/45/2e3445d8a85ff132c343579f2a676174.jpg",
		"pineapple":     "https://i.pinimg.com/736x/4c/66/4e/4c664ee318399b04caede57c2e5a0c76.jpg",
		"watermelon":    "https://i.pinimg.com/1200x/ff/2d/8c/ff2d8c8a66a848409c086fae0f4bad37.jpg",
		"papaya":        "https://i.pinimg.com/1200x/a2/f7/4e/a2f74ed4d7470defd102502df41ea2bc.jpg",
		"strawberry":    "https://i.pinimg.com/736x/be/86/c0/be86c030bd831db9c274965467ce7971.jpg",
		"guava":         "https://i.pinimg.com/1200x/f2/94/c4/f294c43c152a58a2a0581bf26afb4b93.jpg",
		"pomegranate":   "https://i.pinimg.com/736x/2e/36/2a/2e362accd710075ac4a82c47bf8bedfa.jpg",
		"lemon":         "https://i.pinimg.com/736x/ea/d2/ef/ead2ef491f7110336ff0fa3cc9127eca.jpg",
		"cherry":        "https://i.pinimg.com/736x/a6/ef/f4/a6eff4df454e58169547e1ab1a1f9582.jpg",
		"pear":          "https://i.pinimg.com/736x/e6/5c/08/e65c08b54fb2d575dc12a01e031119dc.jpg",
		"kiwi":          "https://i.pinimg.com/1200x/d6/6b/25/d66b257fa3fb12c27b3b0e5f9ebf3885.jpg",
		"melon":         "https://i.pinimg.com/736x/ad/69/69/ad696933e078f9633daa0588e0f5d003.jpg",
		"coconut":       "https://i.pinimg.com/1200x/27/6d/d4/276dd49787cf64a61a64d808ee76d974.jpg",
		"fig":           "https://i.pinimg.com/1200x/d6/e4/cf/d6e4cf0043a5e166b43cb59651181441.jpg",
		"plum":          "https://i.pinimg.com/736x/48/8d/c8/488dc84550443394ea96e1601fae5f3a.jpg",
		"peach":         "https://i.pinimg.com/1200x/6e/b9/62/6eb962f67b5ff57dd230dcafd3d508aa.jpg",
		"dragonfruit":   "https://i.pinimg.com/736x/85/c5/6b/85c56b238c8449df974a55023ad1ea9e.jpg",
		"litchi":        "https://i.pinimg.com/736x/57/74/99/577499786f75973eb176ffb0a4c2ed7a.jpg",
		"custard-apple": "https://i.pinimg.com/1200x/53/f7/3f/53f73fd99c210e13c058aa7d11f75126.jpg",

		// vegetables
		"tomato":      "https://i.pinimg.com/1200x/eb/8e/96/eb8e966fe72dab6db6148c64b404978e.jpg",
		"potato":      "https://i.pinimg.com/1200x/7c/e6/fb/7ce6fb5fad638df14ca07367e36527b4.jpg",
		"carrot":      "https://i.pinimg.com/1200x/01/66/26/0166261fa0018cbb019c459ea8588c06.jpg",
		"spinach":     "https://i.pinimg.com/1200x/6a/f5/8e/6af58eb81351312ec769f0a5a5860fbd.jpg",
		"onion":       "https://i.pinimg.com/736x/bc/01/67/bc016702f08439323f7fdd715da8edf7.jpg",
		"garlic":      "https://i.pinimg.com/1200x/c9/a5/76/c9a57633fe0c4d6f5d4dd88846c48703.jpg",
		"radish":      "https://i.pinimg.com/736x/17/fe/16/17fe16a2827e21a0785c0fd568ed7863.jpg",
		"coriander":   "https://i.pinimg.com/736x/60/3c/3e/603c3e3515959d6c0795e6f34908328b.jpg",
		"cabbage":     "https://i.pinimg.com/1200x/70/91/33/709133b3fa2387dc35c2c539c12c4206.jpg",
		"cauliflower": "https://i.pinimg.com/1200x/f9/a3/a1/f9a3a14f1d0c1ca829f0604c07181e0e.jpg",
		"okra":        "https://i.pinimg.com/736x/cb/79/c2/cb79c24bf22d83b53cf4df2480ea7457.jpg",
		"pumpkin":     "https://i.pinimg.com/1200x/3c/55/20/3c5520a03808841ce47a099a888b6ce4.jpg",
		"cucumber":    "https://i.pinimg.com/736x/47/7a/46/477a4648c42c648a74524106dc608a20.jpg",
		"zucchini":    "https://i.pinimg.com/736x/89/f8/06/89f806a982cd69a0c8dfd835889c6cd4.jpg",
		"beetroot":    "https://i.pinimg.com/736x/f3/1a/53/f31a536e74b2329fa96bd182d3d5ae95.jpg",
		"broccoli":    "https://i.pinimg.com/1200x/14/39/c4/1439c497eaf3e3eed224c5da7c49aea1.jpg",

		// grains
		"wheat":  "https://i.pinimg.com/736x/03/2e/7b/032e7bb8722b5ce3e4352298aede015a.jpg",
		"rice":   "https://i.pinimg.com/1200x/f6/16/59/f616592a533e5af12ca83854f530800e.jpg",
		"corn":   "https://i.pinimg.com/1200x/67/cb/9f/67cb9f8f23d481e8087efddea69079c2.jpg",
		"barley": "https://i.pinimg.com/736x/53/e8/f6/53e8f6d7a3128dd4ec07a59cd0eb8773.jpg",
		"oats":   "https://i.pinimg.com/1200x/3d/71/22/3d71221409b55750ccab87647760a525.jpg",
		"quinoa": "https://i.pinimg.com/736x/4c/bb/34/4cbb3449268900f84a8e9d4428ada8be.jpg",
		"millet": "https://i.pinimg.com/1200x/fa/1a/68/fa1a687a8073f7d836c239992f761ab7.jpg",
		"bajra":  "https://i.pinimg.com/1200x/e9/e2/a8/e9e2a89f4d180d7b0a1bdb0e2e5f1726.jpg",

		// legumes
		"chickpea":    "https://i.pinimg.com/736x/66/d0/a5/66d0a523af5890003bb6d11709e46000.jpg",
		"lentil":      "https://i.pinimg.com/736x/e4/a3/cb/e4a3cb4cb778d47ab39dddfb2aa49255.jpg",
		"soybean":     "https://i.pinimg.com/736x/00/1c/06/001c06055830a77b20649e7c807ddb10.jpg",
		"pea":         "https://i.pinimg.com/1200x/6d/a9/14/6da91415b6b22b78338ca277bdb90361.jpg",
		"kidney-bean": "https://i.pinimg.com/736x/b2/df/52/b2df529aa1e3bbb339908447b1ca45dd.jpg",
		"pigeon-pea":  "https://i.pinimg.com/736x/bc/d9/ad/bcd9ad4b4288e3a545f4b88dd8cdf287.jpg",

		// others
		"sugarcane": "https://i.pinimg.com/1200x/bc/27/92/bc279270d76531cd83436394d7367896.jpg",
		"cotton":    "https://i.pinimg.com/736x/32/3f/66/323f665d3b2e81de82334893728b44f9.jpg",
		"tea":       "https://i.pinimg.com/736x/b3/4a/49/b34a498694a6fadd8628eb104d154c9e.jpg",
		"coffee":    "https://i.pinimg.com/736x/52/76/ff/5276ff3d3923743c62dea0e9e05ddcb8.jpg",

		// spices
		"fenugreek": "https://i.pinimg.com/1200x/bc/27/92/bc279270d76531cd83436394d7367896.jpg",
	}

	cropname = strings.ToLower(cropname)
	cropname = strings.ReplaceAll(cropname, "_", "-")

	if banner, ok := m[cropname]; ok {
		return banner
	}
	return "https://example.com/images/default.jpg"
}
