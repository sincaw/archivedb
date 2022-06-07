import React, {useCallback, useEffect, useState} from 'react';
import {initializeIcons, IStackStyles, IStackTokens, registerIcons, Stack} from '@fluentui/react';
import {Route, Routes, useSearchParams} from "react-router-dom";

import axios from "axios";
import _ from "lodash";
import './App.css';
import Header from "./components/Header";
import Card, {IImageProps, ITweetProps} from "./components/Card";
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

  const [data, updateData] = useState<ITweetProps[]>([]);
  const [searchParams, setSearchParams] = useSearchParams();

  const pageKey = 'page'
  const getCurrentPage = useCallback(
    () => (parseInt(searchParams.get(pageKey)!) || 1),
    [searchParams])

  useEffect(() => {
    if (rightPress) {
      const p = getCurrentPage() + 1
      setSearchParams({page: p.toString()})
    }
  }, [rightPress, getCurrentPage, setSearchParams]);

  useEffect(() => {
    if (leftPress) {
      let p = getCurrentPage() - 1
      p = p < 1 ? 1 : p
      setSearchParams({page: p.toString()})
    }
  }, [leftPress, getCurrentPage, setSearchParams]);

  useEffect(() => {
    window.scrollTo(0, 0)
  }, [data])


  useEffect(() => {
    const limit = 10
    const p = getCurrentPage()

    async function fetch() {
      try {
        const {data} = await axios.get(`/api/list?limit=${limit}&offset=${(p - 1) * limit}`);
        updateData(data.data);
      } catch (e) {
      }
    }

    fetch()
  }, [searchParams, getCurrentPage])

  return <Stack horizontalAlign="center" styles={stackStyles}>
    {data.map(item => <Card key={item.mid} data={item}/>)}
  </Stack>
}