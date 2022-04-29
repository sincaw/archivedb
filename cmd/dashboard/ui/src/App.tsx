import React, {useEffect, useState} from 'react';
import {IStackStyles, IStackTokens, Stack} from '@fluentui/react';
import './App.css';
import Card from "./components/Card";
import axios from "axios";


const stackTokens: IStackTokens = {childrenGap: 5};
const stackStyles: Partial<IStackStyles> = {
  root: {
    width: '960px',
    margin: '0 auto',
    color: '#605e5c',
  },
};


export const App: React.FunctionComponent = () => {
  const [data, updateData] = useState([]);
  const [isError, setError] = useState(false);
  const [isLoading, setLoading] = useState(false);
  useEffect(() => {
    async function fetch() {
      setError(false);
      setLoading(true);
      try {
        const {data} = await axios.get('/list');
        updateData(data.data);
      } catch (e) {
        setError(true);
      }
      setLoading(false);
    }

    fetch()
  }, [])

  return (
    <Stack horizontalAlign="center" verticalAlign="center" styles={stackStyles} tokens={stackTokens}>
      {data.filter(item => {
        if ('retweeted_status' in item) {
          item = item['retweeted_status']
        }
        return item['visible']['list_id'] == 0
      }).map(item => {
        if ('retweeted_status' in item) {
          item = item['retweeted_status']
        }
        let images: string[] = []
        if ('pic_ids' in item) {
          images = (item['pic_ids'] as string[]).map((id): string => {
            if (id in item['pic_infos']) {
              return item['pic_infos'][id]['largest']['url']
            }
            return ''
          }).filter(i => i != '')
        }

        return <Card
          author={item['user']['screen_name']}
          avatar={item['user']['profile_image_url']}
          date={item['created_at']}
          content={item['text_raw']}
          images={images}
        />
      })}
    </Stack>
  );
};
