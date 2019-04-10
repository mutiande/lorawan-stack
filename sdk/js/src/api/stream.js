// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import Oboe from 'oboe'
import Token from '../util/token'

const notify = function (listeners, ...args) {
  listeners.forEach(listener => listener(...args))
}

const unmarshalError = function (error) {
  if (error && error.jsonBody && error.jsonBody.error) {
    return error.jsonBody.error
  }

  return error
}

const id = x => x

const EVENTS = Object.freeze({
  START: 'start',
  EVENT: 'event',
  ERROR: 'error',
})

/**
 * The `Stream` class provides functionality to subscribe and unsubscribe
 * to event streams.
 *
 * @param {string} url - The stream endpoint.
 * @param {string} method  - The HTTP method for the initial request.
 *
 * @example
 * (async () => {
 *    const stream = new Stream('/api/v3/events/applications/app1,app2', "GET");
 *    const conn = await stream.open();
 *    conn
 *      .on('start', () => console.log('conn opened'));
 *      .on('event', message => console.log('received event message', message));
 *      .on('error', error => console.log(error));
 *
 *    stream.close();
 * })()
 */
class Stream {
  constructor (url = '/api/v3/events', method = 'POST') {
    this._channel = null
    this._url = url
    this._method = method
    this._token = new Token().get()
    this._listeners = Object.values(EVENTS)
      .reduce((acc, curr) => ({ ...acc, [curr]: []}), {})
  }

  /**
   *  Opens a new connection.
   *
   * @param {Object} payload - The payload to include with the initial request if
   * it is the `POST\PUT` request, otherwise add the payload to the request url.
   * @returns {Object} - A new subscription object.
   */
  async open (payload) {
    const self = this
    const token = this._token
    const createEventHandler = (name, transform = id) => args => (
      notify(this._listeners[name], transform(args))
    )

    let Authorization = null
    if (typeof token === 'string') {
      Authorization = `Bearer ${token}`
    }

    const tkn = (await token()).access_token
    Authorization = `Bearer ${tkn}`

    const isGet = this._method === 'GET'
    const url = isGet ? `${this._url}/${payload.join(',')}` : this._url
    const body = isGet ? undefined : payload

    this._channel = Oboe({
      url,
      body,
      method: this._method,
      headers: {
        Authorization,
        Accept: 'application/json',
      },
    })

    this._channel
      .start(createEventHandler(EVENTS.START))
      .node('!.result', createEventHandler(EVENTS.EVENT))
      .fail(createEventHandler(EVENTS.ERROR, unmarshalError))

    return {
      on (eventName, callback) {
        if (self._listeners[eventName] === undefined) {
          throw new Error(
            `${eventName} event is not supported. Should be one of: start, error, event`
          )
        }

        self._listeners[eventName].push(callback)

        return this
      },
    }
  }

  /**
   * Closes the active connection.
   */
  close () {
    this._channel.abort()
    this._channel = null
  }
}

export default Stream
