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

export interface IApiFetcherService {
    fetchTokenPrice(): Promise<any | null>;//todo define return type accordingly
    fetchCgPrice(): Promise<any | null>;//todo define return type accordingly
}

class ApiFetcherService implements IApiFetcherService {
    constructor(private readonly apiFetcherService: IApiFetcherService) {
    }

    async fetchTokenPrice(): Promise<string | null> {
        try {
            // const response = await this.fcmClient.send(message);
            // return response;
        } catch (e) {
            this.handleError(e as Error)
        }
        return "change me to response"
    }
    async fetchCgPrice(): Promise<string | null> {
        try {
            // const response = await this.fcmClient.send(message);
            // return response;
        } catch (e) {
            this.handleError(e as Error)
        }
        return "change me to response"
    }

    private handleError(e: Error) {
        console.error("Error sending message:", e);
        //todo implement
    }
}

export class ApiFetcherServiceMock implements IApiFetcherService {
    private tokenPriceReturnValues: any[] = [];
    private cgPriceReturnValues: any[] = [];
    private lastTokenPriceIndex = 0;
    private lastCgPriceIndex = 0;

    async fetchTokenPrice(): Promise<any | null> {
        if (this.tokenPriceReturnValues.length === 0) {
            return null;
        }
        const value = this.tokenPriceReturnValues[this.lastTokenPriceIndex];
        this.lastTokenPriceIndex = (this.lastTokenPriceIndex + 1) % this.tokenPriceReturnValues.length;
        return value;
    }

    async fetchCgPrice(): Promise<any | null> {
        if (this.cgPriceReturnValues.length === 0) {
            return null;
        }
        const value = this.cgPriceReturnValues[this.lastCgPriceIndex];
        this.lastCgPriceIndex = (this.lastCgPriceIndex + 1) % this.cgPriceReturnValues.length;
        return value;
    }

    setTokenPriceReturnValues(tokenPriceReturnValues: any[]) {
        this.tokenPriceReturnValues = tokenPriceReturnValues;
    }

    setCgPriceReturnValues(cgPriceReturnValues: any[]) {
        this.cgPriceReturnValues = cgPriceReturnValues;
    }
}
