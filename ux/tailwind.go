package ux

//
// Tailwind color palette constants. See https://ansi.md.
//

//nolint:goconst // Tailwind palettes intentionally repeat literal color values.
var Tailwind = TailwindColors{
	Slate: Palette{
		C50:  "#f8fafc",
		C100: "#f1f5f9",
		C200: "#e2e8f0",
		C300: "#cad5e2",
		C400: "#90a1b9",
		C500: "#62748e",
		C600: "#45556c",
		C700: "#314158",
		C800: "#1d293d",
		C900: "#0f172b",
		C950: "#020618",
	},
	Gray: Palette{
		C50:  "#f9fafb",
		C100: "#f3f4f6",
		C200: "#e5e7eb",
		C300: "#d1d5dc",
		C400: "#99a1af",
		C500: "#6a7282",
		C600: "#4a5565",
		C700: "#364153",
		C800: "#1e2939",
		C900: "#101828",
		C950: "#030712",
	},
	Zinc: Palette{
		C50:  "#fafafa",
		C100: "#f4f4f5",
		C200: "#e4e4e7",
		C300: "#d4d4d8",
		C400: "#9f9fa9",
		C500: "#71717b",
		C600: "#52525c",
		C700: "#3f3f46",
		C800: "#27272a",
		C900: "#18181b",
		C950: "#09090b",
	},
	Neutral: Palette{
		C50:  "#fafafa",
		C100: "#f5f5f5",
		C200: "#e5e5e5",
		C300: "#d4d4d4",
		C400: "#a1a1a1",
		C500: "#737373",
		C600: "#525252",
		C700: "#404040",
		C800: "#262626",
		C900: "#171717",
		C950: "#0a0a0a",
	},
	Stone: Palette{
		C50:  "#fafaf9",
		C100: "#f5f5f4",
		C200: "#e7e5e4",
		C300: "#d6d3d1",
		C400: "#a6a09b",
		C500: "#79716b",
		C600: "#57534d",
		C700: "#44403b",
		C800: "#292524",
		C900: "#1c1917",
		C950: "#0c0a09",
	},
	Mauve: Palette{
		C50:  "#fafafa",
		C100: "#f3f1f3",
		C200: "#e7e4e7",
		C300: "#d7d0d7",
		C400: "#a89ea9",
		C500: "#79697b",
		C600: "#594c5b",
		C700: "#463947",
		C800: "#2a212c",
		C900: "#1d161e",
		C950: "#0c090c",
	},
	Olive: Palette{
		C50:  "#fbfbf9",
		C100: "#f4f4f0",
		C200: "#e8e8e3",
		C300: "#d8d8d0",
		C400: "#abab9c",
		C500: "#7c7c67",
		C600: "#5b5b4b",
		C700: "#474739",
		C800: "#2b2b22",
		C900: "#1d1d16",
		C950: "#0c0c09",
	},
	Mist: Palette{
		C50:  "#f9fbfb",
		C100: "#f1f3f3",
		C200: "#e3e7e8",
		C300: "#d0d6d8",
		C400: "#9ca8ab",
		C500: "#67787c",
		C600: "#4b585b",
		C700: "#394447",
		C800: "#22292b",
		C900: "#161b1d",
		C950: "#090b0c",
	},
	Taupe: Palette{
		C50:  "#fbfaf9",
		C100: "#f3f1f1",
		C200: "#e8e4e3",
		C300: "#d8d2d0",
		C400: "#aba09c",
		C500: "#7c6d67",
		C600: "#5b4f4b",
		C700: "#473c39",
		C800: "#2b2422",
		C900: "#1d1816",
		C950: "#0c0a09",
	},
	Red: Palette{
		C50:  "#fef2f2",
		C100: "#ffe2e2",
		C200: "#ffc9c9",
		C300: "#ffa2a2",
		C400: "#ff6467",
		C500: "#fb2c36",
		C600: "#e7000b",
		C700: "#c10007",
		C800: "#9f0712",
		C900: "#82181a",
		C950: "#460809",
	},
	Orange: Palette{
		C50:  "#fff7ed",
		C100: "#ffedd4",
		C200: "#ffd6a7",
		C300: "#ffb86a",
		C400: "#ff8904",
		C500: "#ff6900",
		C600: "#f54900",
		C700: "#ca3500",
		C800: "#9f2d00",
		C900: "#7e2a0c",
		C950: "#441306",
	},
	Amber: Palette{
		C50:  "#fffbeb",
		C100: "#fef3c6",
		C200: "#fee685",
		C300: "#ffd230",
		C400: "#ffba00",
		C500: "#fd9a00",
		C600: "#e17100",
		C700: "#bb4d00",
		C800: "#973c00",
		C900: "#7b3306",
		C950: "#461901",
	},
	Yellow: Palette{
		C50:  "#fefce8",
		C100: "#fef9c2",
		C200: "#fff085",
		C300: "#ffdf20",
		C400: "#fcc800",
		C500: "#efb100",
		C600: "#d08700",
		C700: "#a65f00",
		C800: "#894b00",
		C900: "#733e0a",
		C950: "#432004",
	},
	Lime: Palette{
		C50:  "#f7fee7",
		C100: "#ecfcca",
		C200: "#d8f999",
		C300: "#bbf451",
		C400: "#9ae600",
		C500: "#7ccf00",
		C600: "#5ea500",
		C700: "#497d00",
		C800: "#3c6300",
		C900: "#35530e",
		C950: "#192e03",
	},
	Green: Palette{
		C50:  "#f0fdf4",
		C100: "#dcfce7",
		C200: "#b9f8cf",
		C300: "#7bf1a8",
		C400: "#05df72",
		C500: "#00c950",
		C600: "#00a63e",
		C700: "#008236",
		C800: "#016630",
		C900: "#0d542b",
		C950: "#032e15",
	},
	Emerald: Palette{
		C50:  "#ecfdf5",
		C100: "#d0fae5",
		C200: "#a4f4cf",
		C300: "#5ee9b5",
		C400: "#00d492",
		C500: "#00bc7d",
		C600: "#009966",
		C700: "#007a55",
		C800: "#006045",
		C900: "#004f3b",
		C950: "#002c22",
	},
	Teal: Palette{
		C50:  "#f0fdfa",
		C100: "#cbfbf1",
		C200: "#96f7e4",
		C300: "#46ecd5",
		C400: "#00d5be",
		C500: "#00bba7",
		C600: "#009689",
		C700: "#00786f",
		C800: "#005f5a",
		C900: "#0b4f4a",
		C950: "#022f2e",
	},
	Cyan: Palette{
		C50:  "#ecfeff",
		C100: "#cefafe",
		C200: "#a2f4fd",
		C300: "#53eafd",
		C400: "#00d3f2",
		C500: "#00b8db",
		C600: "#0092b8",
		C700: "#007595",
		C800: "#005f78",
		C900: "#104e64",
		C950: "#053345",
	},
	Sky: Palette{
		C50:  "#f0f9ff",
		C100: "#dff2fe",
		C200: "#b8e6fe",
		C300: "#74d4ff",
		C400: "#00bcff",
		C500: "#00a6f4",
		C600: "#0084d1",
		C700: "#0069a8",
		C800: "#00598a",
		C900: "#024a70",
		C950: "#052f4a",
	},
	Blue: Palette{
		C50:  "#eff6ff",
		C100: "#dbeafe",
		C200: "#bedbff",
		C300: "#8ec5ff",
		C400: "#51a2ff",
		C500: "#2b7fff",
		C600: "#155dfc",
		C700: "#1447e6",
		C800: "#193cb8",
		C900: "#1c398e",
		C950: "#162456",
	},
	Indigo: Palette{
		C50:  "#eef2ff",
		C100: "#e0e7ff",
		C200: "#c6d2ff",
		C300: "#a3b3ff",
		C400: "#7c86ff",
		C500: "#615fff",
		C600: "#4f39f6",
		C700: "#432dd7",
		C800: "#372aac",
		C900: "#312c85",
		C950: "#1e1a4d",
	},
	Violet: Palette{
		C50:  "#f5f3ff",
		C100: "#ede9fe",
		C200: "#ddd6ff",
		C300: "#c4b4ff",
		C400: "#a684ff",
		C500: "#8e51ff",
		C600: "#7f22fe",
		C700: "#7008e7",
		C800: "#5d0ec0",
		C900: "#4d179a",
		C950: "#2f0d68",
	},
	Purple: Palette{
		C50:  "#faf5ff",
		C100: "#f3e8ff",
		C200: "#e9d4ff",
		C300: "#dab2ff",
		C400: "#c27aff",
		C500: "#ad46ff",
		C600: "#9810fa",
		C700: "#8200db",
		C800: "#6e11b0",
		C900: "#59168b",
		C950: "#3c0366",
	},
	Fuchsia: Palette{
		C50:  "#fdf4ff",
		C100: "#fae8ff",
		C200: "#f6cfff",
		C300: "#f4a8ff",
		C400: "#ed6aff",
		C500: "#e12afb",
		C600: "#c800de",
		C700: "#a800b7",
		C800: "#8a0194",
		C900: "#721378",
		C950: "#4b004f",
	},
	Pink: Palette{
		C50:  "#fdf2f8",
		C100: "#fce7f3",
		C200: "#fccee8",
		C300: "#fda5d5",
		C400: "#fb64b6",
		C500: "#f6339a",
		C600: "#e60076",
		C700: "#c6005c",
		C800: "#a3004c",
		C900: "#861043",
		C950: "#510424",
	},
	Rose: Palette{
		C50:  "#fff1f2",
		C100: "#ffe4e6",
		C200: "#ffccd3",
		C300: "#ffa1ad",
		C400: "#ff637e",
		C500: "#ff2056",
		C600: "#ec003f",
		C700: "#c70036",
		C800: "#a50036",
		C900: "#8b0836",
		C950: "#4d0218",
	},
}

type TailwindColors struct {
	Slate   Palette
	Gray    Palette
	Zinc    Palette
	Neutral Palette
	Stone   Palette
	Mauve   Palette
	Olive   Palette
	Mist    Palette
	Taupe   Palette
	Red     Palette
	Orange  Palette
	Amber   Palette
	Yellow  Palette
	Lime    Palette
	Green   Palette
	Emerald Palette
	Teal    Palette
	Cyan    Palette
	Sky     Palette
	Blue    Palette
	Indigo  Palette
	Violet  Palette
	Purple  Palette
	Fuchsia Palette
	Pink    Palette
	Rose    Palette
}

type Palette struct {
	C50  string
	C100 string
	C200 string
	C300 string
	C400 string
	C500 string
	C600 string
	C700 string
	C800 string
	C900 string
	C950 string
}
