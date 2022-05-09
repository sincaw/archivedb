import React, {useEffect, useState} from 'react';
import {initializeIcons, IStackStyles, IStackTokens, registerIcons, Stack} from '@fluentui/react';
import {Route, Routes} from "react-router-dom";

import axios from "axios";
import _ from "lodash";
import './App.css';
import Header from "./components/Header";
import Card, {IImageProps} from "./components/Card";
import icons from "./components/Icons";
import {Settings} from "./components/Settings";


const stackTokens: IStackTokens = {};
const mainStackStyles: Partial<IStackStyles> = {
  root: {
    paddingTop: '45px',
  },
};

const stackStyles: Partial<IStackStyles> = {
  root: {
    width: '960px',
    margin: '0 auto',
    color: '#605e5c',
  },
};


const useKeyPress = function (targetKey: string) {
  const [keyPressed, setKeyPressed] = useState(false);

  const downHandler = (ev: KeyboardEvent): any => {
    if (ev.key === targetKey) {
      setKeyPressed(true);
    }
  }

  const upHandler = (ev: KeyboardEvent): any => {
    if (ev.key === targetKey) {
      setKeyPressed(false);
    }
  };

  React.useEffect(() => {
    window.addEventListener("keydown", downHandler);
    window.addEventListener("keyup", upHandler);

    return () => {
      window.removeEventListener("keydown", downHandler);
      window.removeEventListener("keyup", upHandler);
    };
  });

  return keyPressed;
};


export const App: React.FunctionComponent = () => {
  registerIcons(icons);
  initializeIcons();

  return (
    <Stack tokens={stackTokens} styles={mainStackStyles}>
      <Header/>
      <Routes>
        <Route path="/" element={<Weibo/>}/>
        <Route path="settings" element={<Settings/>}/>
      </Routes>
    </Stack>
  );
};


const Weibo: React.FunctionComponent = () => {
  const rightPress = useKeyPress("ArrowRight");
  const leftPress = useKeyPress("ArrowLeft");

  const [data, updateData] = useState([]);
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    if (rightPress) {
      setCurrentPage(p => p + 1)
    }
  }, [rightPress]);

  useEffect(() => {
    if (leftPress && currentPage > 1) {
      setCurrentPage(p => p - 1)
    }
  }, [leftPress, currentPage]);

  useEffect(() => {
    const limit = 10

    async function fetch() {
      try {
        const {data} = await axios.get(`/list?limit=${limit}&offset=${(currentPage - 1) * limit}`);
        updateData(data.data);
      } catch (e) {
      }
    }

    fetch()
  }, [currentPage])
  return <Stack horizontalAlign="center" styles={stackStyles}>
    {data.filter(item => {
      if ('retweeted_status' in item) {
        item = item['retweeted_status']
      }
      return item['visible']['list_id'] === 0
    }).map(item => {
      if ('retweeted_status' in item) {
        item = item['retweeted_status']
      }
      const extraImageKey = "archiveImages"
      let images: IImageProps[] = []
      if ('pic_ids' in item && extraImageKey in item) {
        images = (item['pic_ids'] as string[]).map((id): IImageProps => {
          var ret: IImageProps = {}
          if (id in item[extraImageKey]) {
            const t = item[extraImageKey][id]
            ret.thumbnail = `/resource?key=${t['thumb']}`
            ret.origin = `/resource?key=${t['origin']}`
          }
          return ret
        }).filter(i => i.origin !== '')
      }

      return <Card
        key={_.get(item, 'idstr', '')}
        author={_.get(item, 'user.screen_name', '')}
        avatar={_.get(item, 'user.profile_image_url', '')}
        date={_.get(item, 'created_at', '')}
        content={_.get(item, 'text_raw', '')}
        images={images}
        video={_.get(item, 'video')}
        id={_.get(item, 'idstr')}
      />
    })}
  </Stack>
}