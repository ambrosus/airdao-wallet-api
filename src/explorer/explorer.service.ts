//func (s *service) keepAlive(ctx context.Context) {
// 	loadWatchers := true
// 	for {
// 		var req bytes.Buffer
// 		req.WriteString("{\"id\":\"")
// 		req.WriteString(s.explorerToken)
// 		req.WriteString("\",\"action\":\"init\",\"url\":\"")
// 		req.WriteString(s.callbackUrl)
// 		req.WriteString("\"}")
// 		if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
// 			s.logger.Errorln(err)
// 			time.Sleep(5 * time.Second)
// 			continue
// 		}
//
// 		if loadWatchers {
// 			loadWatchers = false
// 			page := 1
// 			for {
// 				watchers, err := s.repository.GetWatcherList(ctx, bson.M{}, page)
// 				if err != nil {
// 					s.logger.Errorf("keepAlive repository.GetWatcherList error %v", err)
// 					break
// 				}
//
// 				if watchers == nil {
// 					break
// 				}
//
// 				for _, watcher := range watchers {
// 					s.mx.Lock()
// 					s.cachedWatcher[watcher.PushToken] = watcher
// 					s.mx.Unlock()
//
// 					s.setUpStopChanAndStartWatchers(ctx, watcher)
// 				}
//
// 				page++
// 			}
// 		} else {
// 			watchers := make([]*Watcher, 0)
// 			s.mx.Lock()
// 			for _, watcher := range s.cachedWatcher {
// 				watchers = append(watchers, watcher)
// 			}
// 			s.mx.Unlock()
// 			for _, watcher := range watchers {
// 				if watcher.Addresses != nil && len(*watcher.Addresses) > 0 {
// 					var req bytes.Buffer
// 					req.WriteString("{\"id\":\"")
// 					req.WriteString(s.explorerToken)
// 					req.WriteString("\",\"action\":\"subscribe\",\"addresses\":[")
// 					for i, address := range *watcher.Addresses {
// 						if i != 0 {
// 							req.WriteString(",\"")
// 						} else {
// 							req.WriteString("\"")
// 						}
// 						req.WriteString(address.Address)
// 						req.WriteString("\"")
// 					}
// 					req.WriteString("]}")
// 					if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
// 						s.logger.Errorln(err)
// 					}
// 				}
// 			}
// 		}
//
// 		tries := 6
// 		for {
// 			req.Reset()
// 			req.WriteString("{\"id\":\"")
// 			req.WriteString(s.explorerToken)
// 			req.WriteString("\",\"action\":\"check\"}")
// 			if err := s.doRequest(fmt.Sprintf("%s/watch", s.explorerUrl), &req, nil); err != nil {
// 				s.logger.Errorln(err)
// 				if tries != 0 {
// 					tries--
// 					time.Sleep(5 * time.Second)
// 					continue
// 				}
// 				break
// 			}
// 			time.Sleep(30 * time.Second)
// 			tries = 6
// 		}
// 	}
// }
