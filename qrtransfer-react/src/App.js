import React, { useState } from 'react'

const Sender = 0
const Receiver = 1
const RoleUndefined = 2

const roles = [
  { name: '发送者', value: Sender },
  { name: '接收者', value: Receiver },
]

const App = () => {
  const [role, setRole] = useState(RoleUndefined)
  return (
    <>
      <select>
        {roles.map(v => (
          <option value={v.value}>{v.name}</option>
        ))}
      </select>
    </>
  )
}

export default App