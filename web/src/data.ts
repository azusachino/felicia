// Shared fixture data + types for the front-of-house demos (v1 map reader, v2
// memento-first, v3 techo). No I/O — this mirrors the eventual
// Journey -> Visit (derived place) -> Memento model (Dawarich-shaped).

export type Coordinates = [number, number]
export type MementoKind = 'goods' | 'transit' | 'stamp' | 'receipt' | 'souvenir'
export type Lang = 'ja' | 'en' | 'zh'
export type Theme = 'dark' | 'light'

// Japanese is the primary/default language (felicia:decision:jp-first-i18n).
export interface L {
  ja: string
  en: string
  zh: string
}

export interface Station {
  name: string
  ja: string
  coords: Coordinates
}

export interface Memento {
  id: string
  kind: MementoKind
  // The visit (derived place/stay) this memento anchors to. Point mementos sit
  // AT the visit; a transit memento anchors to the endpoint visit it connects
  // (its own movement lives in `transit`). felicia:decision:place-as-derived-visit.
  visitId: string
  title: L
  date: L
  place: L
  vendor: L
  price: string
  coords: Coordinates
  essay: L
  photos: { src: string; caption: L }[]
  transit?: {
    operator: L
    line: L
    from: Station
    to: Station
    fare: string
  }
}

// A place is a DERIVED VISIT — a dwell-time stay detected on the track, the way
// Dawarich / Google Timeline compute them (points -> tracks -> visits @ places ->
// trips). Not stored per se; the demo lists them explicitly so the shape isn't
// misleading. Mementos anchor to a visit; the map clusters by visit.
export interface Visit {
  id: string // stable place key (a coordinate-cluster id in the real system)
  label: L // reverse-geocoded place name
  coords: Coordinates // visit centroid
}

export interface Journey {
  id: string
  title: L
  dates: L
  place: L
  route: Coordinates[] // the track (Dawarich); display route = track ∪ transit legs
  visits: Visit[] // derived stays in travel order; mementos anchor here
  mementos: Memento[]
}

export const stations: Station[] = [
  { name: 'Tokyo', ja: '東京', coords: [139.7671, 35.6812] },
  { name: 'Shibuya', ja: '渋谷', coords: [139.7016, 35.658] },
  { name: 'Asakusa', ja: '浅草', coords: [139.7967, 35.7148] },
  { name: 'Kyoto', ja: '京都', coords: [135.7588, 34.9858] },
  { name: 'Osaka-Umeda', ja: '大阪梅田', coords: [135.4959, 34.7025] },
  { name: 'Sapporo', ja: '札幌', coords: [141.3545, 43.0618] },
  { name: 'Otaru', ja: '小樽', coords: [141.0007, 43.1907] },
  { name: 'Hakata', ja: '博多', coords: [130.4207, 33.5904] },
  { name: 'Kumamoto', ja: '熊本', coords: [130.6889, 32.7845] },
]

export const station = (name: string) => stations.find((item) => item.name === name) ?? stations[0]

export const journeys: Journey[] = [
  {
    id: 'golden-route',
    title: { ja: '日本ゴールデンルート', en: 'Japan Golden Route', zh: '日本黄金路线' },
    dates: { ja: '2026年5月10日〜18日', en: 'May 10–18, 2026', zh: '2026年5月10日–18日' },
    place: { ja: '東京・京都・大阪', en: 'Tokyo · Kyoto · Osaka', zh: '东京·京都·大阪' },
    route: [
      [139.7671, 35.6812],
      [139.7016, 35.658],
      [139.7967, 35.7148],
      [138.389, 35.322],
      [136.8815, 35.1709],
      [135.7588, 34.9858],
      [135.5013, 34.6687],
    ],
    visits: [
      {
        id: 'gr-tokyo',
        label: { ja: '東京', en: 'Tokyo', zh: '东京' },
        coords: [139.7671, 35.6812],
      },
      {
        id: 'gr-kyoto',
        label: { ja: '京都 金閣寺', en: 'Kinkaku-ji, Kyoto', zh: '京都 金阁寺' },
        coords: [135.7292, 35.0394],
      },
      {
        id: 'gr-dotonbori',
        label: { ja: '大阪 道頓堀', en: 'Dotonbori, Osaka', zh: '大阪 道顿堀' },
        coords: [135.5013, 34.6687],
      },
    ],
    mementos: [
      {
        id: 'ginza-line',
        visitId: 'gr-tokyo',
        kind: 'transit',
        title: { ja: '渋谷から浅草へ', en: 'Shibuya to Asakusa', zh: '从涩谷到浅草' },
        date: { ja: '2026年5月12日', en: '2026.05.12', zh: '2026年5月12日' },
        place: { ja: '東京メトロ銀座線', en: 'Tokyo Metro Ginza Line', zh: '东京地铁银座线' },
        vendor: { ja: '東京メトロ', en: 'Tokyo Metro', zh: '东京地铁' },
        price: 'JPY 210',
        coords: [139.7492, 35.6864],
        essay: {
          ja: 'オレンジ色の電車が二つの東京を縫い合わせた。西には渋谷の喧騒、東には浅草の線香の煙。地図の上の一点というより、その一日を読み解けるものにしてくれた一本の線だった。',
          en: 'The orange train stitched two Tokyos together: Shibuya noise in the west, temple smoke in the east. It is less a point on the map than a line that made the day legible.',
          zh: '橙色的电车把两个东京缝在了一起：西边是涩谷的喧嚣，东边是浅草的香火。它与其说是地图上的一个点，不如说是一条让那一天变得清晰的线。',
        },
        photos: [],
        transit: {
          operator: { ja: '東京メトロ', en: 'Tokyo Metro', zh: '东京地铁' },
          line: { ja: '銀座線', en: 'Ginza Line', zh: '银座线' },
          from: station('Shibuya'),
          to: station('Asakusa'),
          fare: 'JPY 210',
        },
      },
      {
        id: 'kinkakuji',
        visitId: 'gr-kyoto',
        kind: 'stamp',
        title: { ja: '金閣寺の御朱印', en: 'Kinkaku-ji Goshuin', zh: '金阁寺御朱印' },
        date: { ja: '2026年5月14日', en: '2026.05.14', zh: '2026年5月14日' },
        place: { ja: '京都 金閣寺', en: 'Kinkaku-ji, Kyoto', zh: '京都 金阁寺' },
        vendor: { ja: '寺務所', en: 'Temple office', zh: '寺务所' },
        price: 'JPY 500',
        coords: [135.7292, 35.0394],
        essay: {
          ja: '金閣は光の中で華やかに輝いていたが、御朱印は静かだった。筆、朱の印、紙。土産を買うというより、その場所のかけらを受け取るような感覚だった。',
          en: 'The gold pavilion was loud in the light, but the stamp was quiet: brush, red seal, paper. It felt less like buying a souvenir and more like receiving a small piece of the place.',
          zh: '金阁在阳光下璀璨夺目，而御朱印却很安静：毛笔、朱印、纸张。与其说是买纪念品，不如说是领受了那个地方的一小片。',
        },
        photos: [
          {
            src: '/kyoto_temple.jpg',
            caption: {
              ja: '朝の色に包まれた金閣寺。',
              en: 'Kinkaku-ji framed by early color.',
              zh: '晨光中的金阁寺。',
            },
          },
        ],
      },
      {
        id: 'fuwamiku',
        visitId: 'gr-dotonbori',
        kind: 'goods',
        title: {
          ja: 'ふわみく マスコットぬいぐるみ',
          en: 'Fuwamiku Mascot Plush',
          zh: 'Fuwamiku 吉祥物玩偶',
        },
        date: { ja: '2026年5月16日', en: '2026.05.16', zh: '2026年5月16日' },
        place: { ja: '大阪 道頓堀', en: 'Dotonbori, Osaka', zh: '大阪 道顿堀' },
        vendor: {
          ja: 'マスコットカフェ&ショップ',
          en: 'Mascot Cafe & Shop',
          zh: '吉祥物咖啡店&商店',
        },
        price: 'JPY 2,400',
        coords: [135.5013, 34.6687],
        essay: {
          ja: '道頓堀のネオンの喧騒を抜けると、カフェはグラスの氷が溶ける音が聞こえるほど静かだった。レジの横にちょこんと座っていた小さなぬいぐるみは、ばかばかしいほど柔らかく、いつのまにかその午後をまるごと家まで連れて帰るものになった。',
          en: 'After the neon crush of Dotonbori, the cafe felt quiet enough to hear the ice settle in our glasses. The tiny plush was sitting beside the register, absurdly soft, and somehow became the thing that carried the whole afternoon home.',
          zh: '穿过道顿堀霓虹的喧嚣，咖啡店安静得能听见杯中冰块融化的声音。收银台旁坐着的那只小玩偶，柔软得近乎荒唐，不知不觉间成了把整个下午都带回家的东西。',
        },
        photos: [
          {
            src: '/osaka_plushie.jpg',
            caption: {
              ja: '後から撮れる「もの」。数ヶ月経っても写真に残せる。',
              en: 'The back-fillable object: still photographable months later.',
              zh: '可以事后再拍的“物件”：几个月后依然能留影。',
            },
          },
        ],
      },
      {
        id: 'dotonbori-takoyaki',
        visitId: 'gr-dotonbori',
        kind: 'goods',
        title: { ja: 'たこ焼きの食べ歩き', en: 'Takoyaki on the Street', zh: '边走边吃的章鱼烧' },
        date: { ja: '2026年5月16日', en: '2026.05.16', zh: '2026年5月16日' },
        place: { ja: '大阪 道頓堀', en: 'Dotonbori, Osaka', zh: '大阪 道顿堀' },
        vendor: { ja: '道頓堀の屋台', en: 'Dotonbori stall', zh: '道顿堀路边摊' },
        price: 'JPY 600',
        coords: [135.5013, 34.6687],
        essay: {
          ja: '同じ道頓堀でも、ぬいぐるみを買った午後とは別の記憶。熱いたこ焼きを手に、川沿いのネオンをただ眺めていた夜。ひとつの場所に、いくつもの記憶が重なっていく。',
          en: 'Same Dotonbori, a different memory from the afternoon of the plush — a night just watching the neon over the canal with a hot box of takoyaki. One place, several memories stacked on top of each other.',
          zh: '同样是道顿堀，却是与买玩偶那个下午不同的记忆——手捧一盒滚烫的章鱼烧，只是望着运河上的霓虹的夜晚。同一个地方，叠着好几段记忆。',
        },
        photos: [],
      },
    ],
  },
  {
    id: 'hokkaido',
    title: { ja: '北海道の冬', en: 'Hokkaido Winter', zh: '北海道的冬天' },
    dates: { ja: '2026年2月3日〜7日', en: 'Feb 3–7, 2026', zh: '2026年2月3日–7日' },
    place: { ja: '札幌・小樽', en: 'Sapporo · Otaru', zh: '札幌·小樽' },
    route: [
      [141.3545, 43.0618],
      [141.0007, 43.1907],
      [142.365, 43.7708],
    ],
    visits: [
      {
        id: 'hk-sapporo',
        label: { ja: '札幌', en: 'Sapporo', zh: '札幌' },
        coords: [141.3545, 43.0618],
      },
      {
        id: 'hk-otaru',
        label: { ja: '小樽', en: 'Otaru', zh: '小樽' },
        coords: [141.0007, 43.1907],
      },
    ],
    mementos: [
      {
        id: 'jr-otaru',
        visitId: 'hk-otaru',
        kind: 'transit',
        title: { ja: '札幌から小樽へ', en: 'Sapporo to Otaru', zh: '从札幌到小樽' },
        date: { ja: '2026年2月4日', en: '2026.02.04', zh: '2026年2月4日' },
        place: { ja: 'JR函館本線', en: 'JR Hakodate Line', zh: 'JR函馆本线' },
        vendor: { ja: 'JR北海道', en: 'JR Hokkaido', zh: 'JR北海道' },
        price: 'JPY 750',
        coords: [141.18, 43.13],
        essay: {
          ja: '雪の海岸線に沿って、快速が小樽へ滑っていく。窓の外は白と灰色ばかりで、車内の暖かさがいっそう際立った。',
          en: 'The rapid slid toward Otaru along a snowbound coast; outside was only white and grey, which made the warmth of the car stand out all the more.',
          zh: '快速列车沿着积雪的海岸线滑向小樽。窗外只有白色与灰色，衬得车厢里格外温暖。',
        },
        photos: [],
        transit: {
          operator: { ja: 'JR北海道', en: 'JR Hokkaido', zh: 'JR北海道' },
          line: { ja: '函館本線', en: 'Hakodate Line', zh: '函馆本线' },
          from: station('Sapporo'),
          to: station('Otaru'),
          fare: 'JPY 750',
        },
      },
      {
        id: 'shiroi-koibito',
        visitId: 'hk-sapporo',
        kind: 'goods',
        title: { ja: '白い恋人', en: 'Shiroi Koibito', zh: '白色恋人' },
        date: { ja: '2026年2月5日', en: '2026.02.05', zh: '2026年2月5日' },
        place: { ja: '札幌', en: 'Sapporo', zh: '札幌' },
        vendor: { ja: '石屋製菓', en: 'Ishiya', zh: '石屋制果' },
        price: 'JPY 1,300',
        coords: [141.3545, 43.0618],
        essay: {
          ja: '空港で最後に買ったのは、やはりこれだった。白い箱の中に、旅の寒さと甘さが一緒に収まっている気がした。',
          en: 'The last thing I bought at the airport was, of course, this. Inside the white box, the cold and the sweetness of the trip seemed to sit together.',
          zh: '在机场最后买的果然还是它。白色的盒子里，仿佛把旅途的寒冷与甜蜜一起装了进去。',
        },
        photos: [],
      },
    ],
  },
  {
    id: 'kyushu',
    title: { ja: '九州横断', en: 'Across Kyushu', zh: '横穿九州' },
    dates: { ja: '2026年4月20日〜24日', en: 'Apr 20–24, 2026', zh: '2026年4月20日–24日' },
    place: { ja: '福岡・熊本', en: 'Fukuoka · Kumamoto', zh: '福冈·熊本' },
    route: [
      [130.4207, 33.5904],
      [130.6889, 32.7845],
      [131.1, 32.88],
    ],
    visits: [
      {
        id: 'ky-hakata',
        label: { ja: '福岡 博多', en: 'Hakata, Fukuoka', zh: '福冈 博多' },
        coords: [130.4207, 33.5904],
      },
      {
        id: 'ky-kumamoto',
        label: { ja: '熊本', en: 'Kumamoto', zh: '熊本' },
        coords: [130.6889, 32.7845],
      },
    ],
    mementos: [
      {
        id: 'kyushu-shinkansen',
        visitId: 'ky-hakata',
        kind: 'transit',
        title: { ja: '博多から熊本へ', en: 'Hakata to Kumamoto', zh: '从博多到熊本' },
        date: { ja: '2026年4月21日', en: '2026.04.21', zh: '2026年4月21日' },
        place: { ja: '九州新幹線', en: 'Kyushu Shinkansen', zh: '九州新干线' },
        vendor: { ja: 'JR九州', en: 'JR Kyushu', zh: 'JR九州' },
        price: 'JPY 3,280',
        coords: [130.55, 33.19],
        essay: {
          ja: '新幹線はわずか三十分あまりで平野を南へ運んだ。速さそのものが、旅の景色を一枚の絵のように圧縮していく。',
          en: 'The shinkansen carried us south across the plain in barely half an hour; the speed itself compressed the landscape into a single picture.',
          zh: '新干线仅用三十多分钟便把我们送过平原一路向南。速度本身，把旅途的风景压缩成了一幅画。',
        },
        photos: [],
        transit: {
          operator: { ja: 'JR九州', en: 'JR Kyushu', zh: 'JR九州' },
          line: { ja: '九州新幹線', en: 'Kyushu Shinkansen', zh: '九州新干线' },
          from: station('Hakata'),
          to: station('Kumamoto'),
          fare: 'JPY 3,280',
        },
      },
      {
        id: 'kumamon',
        visitId: 'ky-kumamoto',
        kind: 'goods',
        title: { ja: 'くまモン ぬいぐるみ', en: 'Kumamon Plush', zh: '熊本熊玩偶' },
        date: { ja: '2026年4月22日', en: '2026.04.22', zh: '2026年4月22日' },
        place: { ja: '熊本', en: 'Kumamoto', zh: '熊本' },
        vendor: { ja: 'くまモンスクエア', en: 'Kumamon Square', zh: '熊本熊广场' },
        price: 'JPY 1,980',
        coords: [130.6889, 32.7845],
        essay: {
          ja: '熊本のどこにでもいる黒い熊が、いちばん小さな姿で棚に並んでいた。旅の記憶は、こういう軽いものに限って重い。',
          en: "Kumamoto's black bear, everywhere in the city, lined the shelf in its smallest form. Travel memories are heaviest, oddly, in the lightest objects.",
          zh: '在熊本随处可见的那只黑熊，以最小的模样排在货架上。旅行的记忆，偏偏寄托在这样轻巧的东西上最沉。',
        },
        photos: [],
      },
    ],
  },
]

// Chronological timeline is the index (liuaaron-aligned): sort each journey by date.
for (const journey of journeys) {
  journey.mementos.sort((a, b) => a.date.en.localeCompare(b.date.en))
}

export const kindLabel: Record<MementoKind, L> = {
  transit: { ja: '交通', en: 'Transit', zh: '交通' },
  stamp: { ja: '御朱印', en: 'Stamp', zh: '御朱印' },
  goods: { ja: 'グッズ', en: 'Goods', zh: '周边' },
  receipt: { ja: 'レシート', en: 'Receipt', zh: '收据' },
  souvenir: { ja: 'おみやげ', en: 'Souvenir', zh: '纪念品' },
}

export const uiText = {
  journeys: { ja: '旅の記録', en: 'Journeys', zh: '旅程' },
  all: { ja: 'すべて表示', en: 'View all', zh: '查看全部' },
  story: { ja: '物語', en: 'The Story', zh: '故事' },
}

// v2 (memento-first): a flat "greatest-hits" shelf across every journey, each
// item carrying its parent journey for context in the preview carousel.
export interface MementoCard {
  memento: Memento
  journey: Journey
}

export const allMementos: MementoCard[] = journeys.flatMap((journey) =>
  journey.mementos.map((memento) => ({ memento, journey })),
)
