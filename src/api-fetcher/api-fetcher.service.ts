// func (s *service) CGWatch(ctx context.Context) {
//     for {
//         var cgData *CGData
//         if err := s.doRequest("https://api.coingecko.com/api/v3/coins/amber/market_chart?vs_currency=usd&days=30", nil, &cgData); err != nil {
//         s.logger.Errorln(err)
//     }
//
//     s.cachedCgPrice = cgData.Prices
//
//     time.Sleep(12 * time.Hour)
// }
// }

// func (s *service) ApiPriceWatch(ctx context.Context) {
//     for {
//         var priceData *PriceData
//         if err := s.doRequest(s.tokenPriceUrl, nil, &priceData); err != nil {
//         s.logger.Errorln(err)
//     }
//
//     if priceData != nil {
//         s.cachedPrice = priceData.Data.PriceUSD
//     }
//
//     time.Sleep(5 * time.Minute)
// }
// }
