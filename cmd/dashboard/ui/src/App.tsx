import React, {useCallback, useEffect, useState} from 'react';
import {Route, Routes, useSearchParams} from 'react-router-dom';
import {Layout, Menu, Pagination} from 'antd';

import axios from "axios";
import './App.css';
import Tweet, {ITweetProps} from "./components/Tweet";
import {Settings} from "./components/Settings";

const {Header, Content, Footer} = Layout;

export const App: React.FunctionComponent = () => {
  return (
    <Layout>
      <Header style={{position: 'fixed', zIndex: 1, width: '100%'}}>
        <Menu
          theme="dark"
          mode="horizontal"
          defaultSelectedKeys={['1']}
          items={new Array(3).fill(null).map((_, index) => ({
            key: String(index + 1),
            label: `nav ${index + 1}`,
          }))}
        />
      </Header>
      <Content className="site-layout" style={{padding: '0 50px', marginTop: 64}}>
        <div style={{marginTop: 10}}>
          <Routes>
            <Route path="/" element={<Weibo/>}/>
            <Route path="settings" element={<Settings/>}/>
          </Routes>
        </div>
      </Content>
      <Footer style={{textAlign: 'center'}}>ArchiveDB ©2022</Footer>
    </Layout>
  );
};

const Weibo: React.FunctionComponent = () => {
  const [data, updateData] = useState<ITweetProps[]>([]);
  const [searchParams, setSearchParams] = useSearchParams();

  const pageKey = 'page'
  const getCurrentPage = useCallback(
    () => (parseInt(searchParams.get(pageKey)!) || 1),
    [searchParams]
  )

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

  return <div style={{margin: '0 auto', width: 820}}>
    {data.map(item => <Tweet key={item.mid} data={item}/>)}
    <Pagination
      defaultCurrent={getCurrentPage()}
      total={500}
      showSizeChanger={false}
      style={{margin: '0 auto', width: 500}}
      onChange={(p, _)=>setSearchParams({page: p.toString()})}
    />
  </div>
}
