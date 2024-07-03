import { singleton } from "tsyringe";
import { HistoricalNotificationsRepository } from "./historical-notifications.repository";


@singleton()
export class HistoricalNotificationsService {
    constructor(private readonly repository: HistoricalNotificationsRepository) {
    }

    async getHistoricalNotifications(watcherId: string) {
        return this.repository.getHistoricalNotifications(watcherId);
    }
}